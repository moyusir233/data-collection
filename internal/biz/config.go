package biz

type ConfigUsecase struct {
}
type ConfigRepo interface {
	SaveDataToHash(key, field string, value []byte) error
}

func NewConfigUsecase() *ConfigUsecase {
	return nil
}
