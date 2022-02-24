package biz

import "time"

type WarningDetectUsecase struct {
}
type WarningDetectRepo interface {
	SaveDataToZset(key, score string, value []byte) error
	SaveDataToTs(key, stamp time.Time, value []byte) error
}

func NewWarningDetectUsecase() *WarningDetectUsecase {
	return nil
}
