package im_err

import (
	"github.com/lizuowang/gim/pkg/logger"
	"go.uber.org/zap"
)

// 标准日志库
// Standard Log Library
type errLog struct{}

func NewErrLog() *errLog {
	return &errLog{}
}

// Error 打印错误日志
// Printing the error log
func (l *errLog) Error(v ...any) {
	logger.L.Error("gim_sys_error", zap.Any("error", v))
}
