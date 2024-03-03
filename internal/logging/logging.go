package logging

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Logger is the interface to our internal logger.
type Logger interface {
	Debug(msg string, kvpairs ...interface{}) //内部调试日志信息
	Info(msg string, kvpairs ...interface{})  //一般操作信息
	Error(msg string, kvpairs ...interface{}) //错误信息
	SetField(key string, val interface{})     //设置键值对，记录日志消息外的上下文状态
	PushFields()                              //压入上下文状态
	PopFields()                               //推出
}

// LogrusLogger is a thread-safe logger whose properties persist and can be modified.
type LogrusLogger struct { //日志记录器，NewLogrusLogger使用LogrusLogger生成实例化Logger
	mtx             sync.Mutex               //同步互斥
	logger          *logrus.Entry            //日志条目
	ctx             string                   //交易
	fields          map[string]interface{}   //键值对存储
	pushedFieldSets []map[string]interface{} //存储已压入的键值对
}

// NoopLogger implements Logger, but does nothing.
type NoopLogger struct{}

// LogrusLogger implements Logger
var _ Logger = (*LogrusLogger)(nil)
var _ Logger = (*NoopLogger)(nil)

//
// LogrusLogger
//

// NewLogrusLogger will instantiate a logger with the given context.
func NewLogrusLogger(ctx string, kvpairs ...interface{}) Logger { //接收ctx和kvpairs构造Logger
	var logger *logrus.Entry
	if len(ctx) > 0 {
		logger = logrus.WithField("ctx", ctx)
	} else {
		logger = logrus.NewEntry(logrus.New())
	}
	return &LogrusLogger{
		logger:          logger,
		ctx:             ctx,
		fields:          serializeKVPairs(kvpairs),
		pushedFieldSets: []map[string]interface{}{},
	}
}

func (l *LogrusLogger) withFields() *logrus.Entry {
	if len(l.fields) > 0 {
		return l.logger.WithFields(l.fields) //返回带ctx的字段
	}
	return l.logger //返回原字段
}

func serializeKVPairs(kvpairs ...interface{}) map[string]interface{} { //存储键值对
	res := make(map[string]interface{})
	if (len(kvpairs) % 2) == 0 {
		for i := 0; i < len(kvpairs); i += 2 {
			res[kvpairs[i].(string)] = kvpairs[i+1]
		}
	}
	return res
}

func (l *LogrusLogger) withKVPairs(kvpairs ...interface{}) *logrus.Entry {
	fields := serializeKVPairs(kvpairs...)
	if len(fields) > 0 {
		return l.withFields().WithFields(fields) //返回带键值对的字段
	}
	return l.withFields() //返回原字段
}

func (l *LogrusLogger) Debug(msg string, kvpairs ...interface{}) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.withKVPairs(kvpairs...).Debugln(msg)
}

func (l *LogrusLogger) Info(msg string, kvpairs ...interface{}) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.withKVPairs(kvpairs...).Infoln(msg)
}

func (l *LogrusLogger) Error(msg string, kvpairs ...interface{}) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.withKVPairs(kvpairs...).Errorln(msg)
}

func (l *LogrusLogger) SetField(key string, val interface{}) {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.fields[key] = val
}

func (l *LogrusLogger) PushFields() {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	l.pushedFieldSets = append(l.pushedFieldSets, l.fields)
}

func (l *LogrusLogger) PopFields() {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	pfsLen := len(l.pushedFieldSets)
	if pfsLen > 0 {
		l.fields = l.pushedFieldSets[pfsLen-1]
		l.pushedFieldSets = l.pushedFieldSets[:pfsLen-1]
	}
}

//
// NoopLogger
//

// NewNoopLogger will instantiate a logger that does nothing when called.
func NewNoopLogger() Logger {
	return &NoopLogger{}
}

func (l *NoopLogger) Debug(msg string, kvpairs ...interface{}) {}
func (l *NoopLogger) Info(msg string, kvpairs ...interface{})  {}
func (l *NoopLogger) Error(msg string, kvpairs ...interface{}) {}
func (l *NoopLogger) SetField(key string, val interface{})     {}
func (l *NoopLogger) PushFields()                              {}
func (l *NoopLogger) PopFields()                               {}
