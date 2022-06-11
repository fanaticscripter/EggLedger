package api

import (
	"context"

	"github.com/fanaticscripter/EggLedger/ei"
)

func RequestFirstContactRawPayloadWithContext(ctx context.Context, playerId string) ([]byte, error) {
	req := &ei.EggIncFirstContactRequest{
		Rinfo:         NewBasicRequestInfo(playerId),
		EiUserId:      &playerId,
		DeviceId:      sptr("EggLedger"), // This is actually bot_name for /ei/bot_first_contact.
		ClientVersion: u32ptr(ClientVersion),
		Platform:      Platform.Enum(),
	}
	payload, err := RequestRawPayloadWithContext(ctx, "/ei/bot_first_contact", req)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func DecodeFirstContactPayload(payload []byte) (*ei.EggIncFirstContactResponse, error) {
	msg := &ei.EggIncFirstContactResponse{}
	err := DecodeAPIResponse(_apiPrefix+"/ei/bot_first_contact", payload, msg, false)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func RequestCompleteMissionRawPayloadWithContext(ctx context.Context, playerId string, missionId string) ([]byte, error) {
	req := &ei.MissionRequest{
		Rinfo:    NewBasicRequestInfo(playerId),
		EiUserId: &playerId,
		Info: &ei.MissionInfo{
			Identifier: &missionId,
		},
		ClientVersion: u32ptr(ClientVersion),
	}
	payload, err := RequestRawPayloadWithContext(ctx, "/ei_afx/complete_mission", req)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func DecodeCompleteMissionPayload(payload []byte) (*ei.CompleteMissionResponse, error) {
	msg := &ei.CompleteMissionResponse{}
	err := DecodeAPIResponse(_apiPrefix+"/ei_afx/complete_mission", payload, msg, true)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
