package baseError

import (
	"fmt"
	"github.com/pkg/errors"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
)

var sourceDir string

func init() {
	_, file, _, _ := runtime.Caller(0)
	// compatible solution to get gorm source directory with various operating systems
	sourceDir = getSourceDir(file)
}
func getSourceDir(file string) string {
	dir := filepath.Dir(file)
	dir = filepath.Dir(dir)

	s := filepath.Dir(dir)
	if filepath.Base(s) != "github.com" {
		s = dir
	}
	return filepath.ToSlash(s) + "/"
}

func IsBaseError(err error) bool {
	return reflect.TypeOf(err).String() == "*baseError.Error"
}

func IsSystemError(err error) bool {
	return reflect.TypeOf(err).String() == "*baseError.Error" && err.(*Error).System
}

func IsNotSystemError(err error) bool {
	return reflect.TypeOf(err).String() == "*baseError.Error" && !err.(*Error).System
}

type Error struct {
	Code   string `json:"code"`
	Msg    string `json:"msg"`
	System bool   `json:"-"`
	Chain  string `json:"-"`
	cause  error
	*stack
}

func (b *Error) WithSystem() *Error {
	b.System = true
	return b
}

func (b *Error) WithChain(chain ...string) *Error {
	b.Chain = strings.Join(chain, "<-")
	return b
}

func (b *Error) WithCause(cause error) *Error {
	b.cause = cause
	return b
}

func (b *Error) WithStack() *Error {
	b.stack = callers(3, 3)
	return b
}

func (b *Error) WithStackDepth(stackDepth int) *Error {
	if stackDepth <= 0 {
		stackDepth = 1
	}
	b.stack = callers(3, stackDepth)
	return b
}

func (b *Error) Error() string {
	return fmt.Sprintf("[%s] %s", b.Code, b.Msg)
}

func (b *Error) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			io.WriteString(s, b.Error())
			if b.stack != nil {
				b.stack.Format(s, verb)
			}
			if b.cause != nil {
				fmt.Fprintf(s, "\n---cause---\n%+v", b.cause)
			}
			return
		}
		fallthrough
	case 's':
		io.WriteString(s, b.Error())
	case 'q':
		fmt.Fprintf(s, "%q", b.Error())
	}
}

func (b *Error) Stack() *stack {
	return b.stack
}

func (b *Error) Cause() error {
	return b.cause
}

func (b *Error) Unwrap() error {
	return b.cause
}

type Option func(*ErrorOption)
type ErrorOption struct {
	system     bool
	chain      []string
	cause      error
	stackDepth int
	formatArgs []any
}

func WithSystem() Option {
	return func(opts *ErrorOption) {
		opts.system = true
	}
}

func WithChain(chain ...string) Option {
	return func(opts *ErrorOption) {
		opts.chain = chain
	}
}

func WithCause(cause error) Option {
	return func(opts *ErrorOption) {
		opts.cause = cause
	}
}

func WithStack() Option {
	return func(opts *ErrorOption) {
		opts.stackDepth = 3
	}
}

func WithStackDepth(stackDepth int) Option {
	return func(opts *ErrorOption) {
		if stackDepth <= 0 {
			stackDepth = 1
		}
		opts.stackDepth = stackDepth
	}
}

func WithFormatArgs(formatArgs ...any) Option {
	return func(opts *ErrorOption) {
		opts.formatArgs = formatArgs
	}
}

func New(code string, msg string, opts ...Option) *Error {
	e := &Error{Code: code, Msg: msg}
	return ApplyOption(e, opts...)
}

func Init(err *Error, opts ...Option) *Error {
	e := &Error{Code: err.Code, Msg: err.Msg, System: err.System}
	return ApplyOption(e, opts...)
}

func WrapCode(code string, err error, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	e := &Error{Code: code, Msg: err.Error(), cause: err}
	return ApplyOption(e, opts...)
}

func ApplyOption(err *Error, opts ...Option) *Error {
	if len(opts) == 0 {
		return err
	}
	errOpt := &ErrorOption{}
	for _, apply := range opts {
		if apply != nil {
			apply(errOpt)
		}
	}
	if errOpt.system {
		err.System = true
	}
	if len(errOpt.chain) > 0 {
		err.Chain = strings.Join(errOpt.chain, "<-")
	}
	if errOpt.cause != nil {
		err.cause = errOpt.cause
	}
	if errOpt.stackDepth != 0 {
		err.stack = callers(3, errOpt.stackDepth)
	}
	if len(errOpt.formatArgs) > 0 {
		err.Msg = fmt.Sprintf(err.Msg, errOpt.formatArgs...)
	}
	return err
}

type stack []uintptr

func (s *stack) Format(st fmt.State, verb rune) {
	switch verb {
	case 'v':
		switch {
		case st.Flag('+'):
			for _, pc := range *s {
				f := errors.Frame(pc)
				fmt.Fprintf(st, "\n%+v", f)
			}
		}
	}
}
func (s *stack) StackTrace() errors.StackTrace {
	f := make([]errors.Frame, len(*s))
	for i := 0; i < len(f); i++ {
		f[i] = errors.Frame((*s)[i])
	}
	return f
}

func callers(skip int, depth int) *stack {
	var s = skip
	for i := skip; i < 15; i++ {
		_, file, _, ok := runtime.Caller(i)
		if ok && (!strings.HasPrefix(file, sourceDir) || strings.HasSuffix(file, "_test.go")) {
			s = i + 1
			break
		}
	}
	pcs := make([]uintptr, depth)
	n := runtime.Callers(s, pcs[:])
	var st stack = pcs[0:n]
	return &st
}
