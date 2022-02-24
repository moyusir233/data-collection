package biz

import "time"

type WarningDetectUsecase struct {
}
type WarningDetectRepo interface {
	SaveDataToZset(key string, score float64, value []byte) error
	SaveDataToTs(key string, stamp time.Time, value []byte) error
}

func NewWarningDetectUsecase() *WarningDetectUsecase {
	return nil
}
