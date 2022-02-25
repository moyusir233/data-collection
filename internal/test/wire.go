//go:build wireinject
// +build wireinject

// The build tag makes sure the stub is not built in the final build.

package test

import (
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/dataCollection/internal/data"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
)

// InitConfigUsecase 测试用的辅助函数
func InitConfigUsecase(*conf.Data, log.Logger) (*biz.ConfigUsecase, func(), error) {
	panic(wire.Build(data.ProviderSet, biz.NewConfigUsecase))
}

// InitWarningDetectUsecase 测试用的辅助函数
func InitWarningDetectUsecase(*conf.Data, log.Logger) (*biz.WarningDetectUsecase, func(), error) {
	panic(wire.Build(data.ProviderSet, biz.NewWarningDetectUsecase))
}
