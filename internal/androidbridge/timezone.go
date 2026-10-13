package androidbridge

import "time"

func setLocalTimezone(name string) error {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return err
	}
	time.Local = loc
	return nil
}
