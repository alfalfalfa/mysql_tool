package errors

import (
	"errors"
	"runtime"
	"strconv"
	"strings"
)

type errorDetail struct {
	err        error
	stackTrace []string
}

const defaultCallerDepth = 2
const printdir = 2

func newErrorDetail(callerDepth int, err error) *errorDetail {
	skipCount := defaultCallerDepth + callerDepth
	res := &errorDetail{err: err}

	//スタックトレース生成
	res.stackTrace = make([]string, 0, 10)

	goroot := runtime.GOROOT()
	//同パッケージ内の呼び出しをスキップ
	for skip := skipCount; ; skip++ {
		_, file, line, _ := runtime.Caller(skip)
		//mainまで遡る
		if strings.HasPrefix(file, goroot) {
			break
		}

		var buf = make([]byte, 0, 128)
		//printdirディレクトリ上まで出力
		file = GetShortPath(file, printdir)
		buf = append(buf, file...)
		buf = append(buf, ":"...)
		buf = append(buf, strconv.Itoa(line)...)
		res.stackTrace = append(res.stackTrace, string(buf))
	}

	return res
}

func GetShortPath(file string, dir int) string {
	short := file
	c := 0
	for i := len(file) - 1; i > 0; i-- {
		if file[i] == '/' {
			c++
			if c > dir {
				short = file[i+1:]
				break
			}
		}
	}
	return short
}

//スタックトレース込でエラー出力
func (e *errorDetail) Error() string {
	var buf = make([]byte, 0, 128)
	//buf = append(buf, "error:"...)
	buf = append(buf, e.err.Error()...)
	buf = append(buf, "\r\n"...)
	for _, v := range e.stackTrace {
		buf = append(buf, "\t"...)
		//buf = append(buf, "    "...)
		buf = append(buf, v...)
		buf = append(buf, "\r\n"...)
	}
	return string(buf)
}

func Wrap(err error) error {
	if err == nil {
		return err
	}
	//スタックトレース生成は一回だけ
	if _, ok := err.(*errorDetail); ok {
		return err
	}
	//if _, ok := err.(errorDetail); ok {
	//	return &err
	//}
	return newErrorDetail(0, err)
}

func WrapWithDepth(callerDepth int, err error) error {
	if err == nil {
		return err
	}
	//スタックトレース生成は一回だけ
	if _, ok := err.(*errorDetail); ok {
		return err
	}
	//if _, ok := err.(errorDetail); ok {
	//	return &err
	//}
	return newErrorDetail(callerDepth, err)
}

func New(msg string) error {
	return newErrorDetail(0, errors.New(msg))
}
