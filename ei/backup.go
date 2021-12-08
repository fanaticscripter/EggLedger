package ei

import (
	"github.com/pkg/errors"
)

func (fc *EggIncFirstContactResponse) Validate() error {
	if fc.GetErrorCode() > 0 {
		return errors.Errorf("/ei/first_contact: error_code %d", fc.GetErrorCode())
	}
	if fc.Backup == nil || fc.GetBackup().Game == nil {
		return errors.New("backup is empty")
	}
	if fc.GetBackup().Settings == nil {
		return errors.New("backup settings is empty")
	}
	if fc.GetBackup().ArtifactsDb == nil {
		return errors.New("backup has empty artifacts database")
	}
	return nil
}
