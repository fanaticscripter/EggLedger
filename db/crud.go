package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/fanaticscripter/EggLedger/api"
	"github.com/fanaticscripter/EggLedger/ei"
)

func InsertBackup(playerId string, timestamp float64, payload []byte, minimumTimeSinceLastEntry time.Duration) error {
	action := fmt.Sprintf("insert backup for player %s into database", playerId)
	compressedPayload, err := compress(payload)
	if err != nil {
		return errors.Wrap(err, action)
	}
	return transact(action, func(tx *sql.Tx) error {
		var previousTimestamp float64
		if minimumTimeSinceLastEntry.Seconds() > 0 {
			row := tx.QueryRow(`SELECT backed_up_at FROM backup
			WHERE player_id = ?
			ORDER BY backed_up_at DESC LIMIT 1;`,
				playerId)
			err := row.Scan(&previousTimestamp)
			switch {
			case err == sql.ErrNoRows:
				// No stored backup
			case err != nil:
				return err
			}
		}
		timeSinceLastEntry := time.Duration(timestamp-previousTimestamp) * time.Second
		if timeSinceLastEntry < minimumTimeSinceLastEntry {
			log.Infof("%s: %s since last recorded backup, ignoring", playerId, timeSinceLastEntry)
			return nil
		}
		_, err = tx.Exec(`INSERT INTO
			backup(player_id, backed_up_at, payload, payload_authenticated)
			VALUES (?, ?, ?, FALSE);`,
			playerId, timestamp, compressedPayload)
		if err != nil {
			return err
		}
		return nil
	})
}

func InsertCompleteMission(playerId string, missionId string, startTimestamp float64, completePayload []byte) error {
	action := fmt.Sprintf("insert mission %s for player %s into database", missionId, playerId)
	compressedPayload, err := compress(completePayload)
	if err != nil {
		return errors.Wrap(err, action)
	}
	return transact(action, func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO
			mission(player_id, mission_id, start_timestamp, complete_payload)
			VALUES (?, ?, ?, ?);`,
			playerId, missionId, startTimestamp, compressedPayload)
		if err != nil {
			return err
		}
		return nil
	})
}

// RetrieveCompleteMission returns the stored CompleteMissionResponse, or nil if not found.
func RetrieveCompleteMission(playerId string, missionId string) (*ei.CompleteMissionResponse, error) {
	action := fmt.Sprintf("retrieve mission %s for player %s from database", missionId, playerId)
	var startTimestamp float64
	var compressedPayload []byte
	err := transact(action, func(tx *sql.Tx) error {
		row := tx.QueryRow(`SELECT start_timestamp, complete_payload FROM mission
			WHERE player_id = ? AND mission_id = ?;`,
			playerId, missionId)
		err := row.Scan(&startTimestamp, &compressedPayload)
		switch {
		case err == sql.ErrNoRows:
			// No such mission
		case err != nil:
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if compressedPayload == nil {
		return nil, nil
	}
	completePayload, err := decompress(compressedPayload)
	if err != nil {
		return nil, errors.Wrap(err, action)
	}
	m, err := api.DecodeCompleteMissionPayload(completePayload)
	if err != nil {
		return nil, errors.Wrap(err, action)
	}
	// /ei_afx/complete_mission response leaves out start_time_derived, so we
	// have to manually attach it.
	m.Info.StartTimeDerived = &startTimestamp
	return m, nil
}

// RetrievePlayerCompleteMissions retrieves stored completed missions for a
// player, in chronological order.
func RetrievePlayerCompleteMissions(playerId string) ([]*ei.CompleteMissionResponse, error) {
	action := fmt.Sprintf("retrieve complete missions for player %s from database", playerId)
	var count int
	var startTimestamps []float64
	var compressedPayloads [][]byte
	err := transact(action, func(tx *sql.Tx) error {
		rows, err := tx.Query(`SELECT start_timestamp, complete_payload FROM mission
			WHERE player_id = ?
			ORDER BY start_timestamp;`, playerId)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var startTimestamp float64
			var compressedPayload []byte
			if err := rows.Scan(&startTimestamp, &compressedPayload); err != nil {
				return err
			}
			count++
			startTimestamps = append(startTimestamps, startTimestamp)
			compressedPayloads = append(compressedPayloads, compressedPayload)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	var missions []*ei.CompleteMissionResponse
	for i := 0; i < count; i++ {
		completePayload, err := decompress(compressedPayloads[i])
		if err != nil {
			return nil, errors.Wrap(err, action)
		}
		m, err := api.DecodeCompleteMissionPayload(completePayload)
		if err != nil {
			return nil, errors.Wrap(err, action)
		}
		m.Info.StartTimeDerived = &startTimestamps[i]
		missions = append(missions, m)
	}
	return missions, nil
}

// RetrievePlayerCompleteMissionIds retrieves IDs of stored completed missions
// for a player, in chronological order.
func RetrievePlayerCompleteMissionIds(playerId string) ([]string, error) {
	action := fmt.Sprintf("retrieve complete mission ids for player %s from database", playerId)
	var missionIds []string
	err := transact(action, func(tx *sql.Tx) error {
		rows, err := tx.Query(`SELECT mission_id FROM mission
			WHERE player_id = ?
			ORDER BY start_timestamp;`, playerId)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var missionId string
			if err := rows.Scan(&missionId); err != nil {
				return err
			}
			missionIds = append(missionIds, missionId)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return missionIds, nil
}

func transact(description string, txFunc func(*sql.Tx) error) (err error) {
	tx, err := _db.Begin()
	if err != nil {
		return errors.Wrap(err, description)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()
			err = errors.Wrap(err, description)
		} else {
			err = tx.Commit()
			if err != nil {
				err = errors.Wrap(err, description)
			}
		}
	}()
	err = txFunc(tx)
	return err
}
