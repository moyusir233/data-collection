package biz

type WarningDetectUsecase struct {
}
type WarningDetectRepo interface {
	SaveDataToZset(key string, score float64, value []byte) error
	SaveDataToTs(key string, stamp int64, value []byte) error
}

func NewWarningDetectUsecase() *WarningDetectUsecase {
	return nil
}
