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
	Code   string   `json:"code"`
	Msg    string   `json:"msg"`
	System bool     `json:"-"`
	Chain  []string `json:"-"`
	cause  error
	*stack
}

func (b *Error) WithCode(code string) *Error {
	b.Code = code
	return b
}

func (b *Error) WithMsg(msg string) *Error {
	b.Msg = msg
	return b
}

func (b *Error) WithMsgArgs(args ...any) *Error {
	b.Msg = fmt.Sprintf(b.Msg, args...)
	return b
}

func (b *Error) WithMsgFormat(format string, args ...any) *Error {
	b.Msg = fmt.Sprintf(format, args...)
	return b
}

func (b *Error) WithSystem() *Error {
	b.System = true
	return b
}

func (b *Error) WithChain(chain ...string) *Error {
	b.Chain = append(b.Chain, chain...)
	return b
}

func (b *Error) WithCause(cause error) *Error {
	b.cause = cause
	return b
}

func (b *Error) WithStack(depth ...int) *Error {
	var d = 3
	if len(depth) > 0 && depth[0] > 0 {
		d = depth[0]
	}
	b.stack = callers(3, d)
	return b
}

func (b *Error) Clone(opts ...Option) *Error {
	return Clone(b, opts...)
}

func (b *Error) Error() string {
	if b.Code != "" {
		return fmt.Sprintf("[%s] %s", b.Code, b.Msg)
	} else {
		return b.Msg
	}
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
	code    string
	msg     string
	system  bool
	chain   []string
	cause   error
	depth   int
	msgArgs []any
}

func WithCode(code string) Option {
	return func(opts *ErrorOption) {
		opts.code = code
	}
}

func WithMsg(msg string) Option {
	return func(opts *ErrorOption) {
		opts.msg = msg
	}
}
func WithMsgFormat(format string, args ...any) Option {
	return func(opts *ErrorOption) {
		opts.msg = fmt.Sprintf(format, args...)
	}
}

func WithMsgArgs(args ...any) Option {
	return func(opts *ErrorOption) {
		opts.msgArgs = args
	}
}

func WithSystem() Option {
	return func(opts *ErrorOption) {
		opts.system = true
	}
}

func WithChain(chain ...string) Option {
	return func(opts *ErrorOption) {
		opts.chain = append(opts.chain, chain...)
	}
}

func WithCause(cause error) Option {
	return func(opts *ErrorOption) {
		opts.cause = cause
	}
}

func WithStack(depth ...int) Option {
	return func(opts *ErrorOption) {
		var d = 3
		if len(depth) > 0 && depth[0] > 0 {
			d = depth[0]
		}
		opts.depth = d
	}
}

func New(msg string, opts ...Option) *Error {
	e := &Error{Msg: msg}
	return ApplyOption(e, opts...)
}

func NewSystem(msg string, opts ...Option) *Error {
	e := &Error{Msg: msg, System: true}
	return ApplyOption(e, opts...)
}

func NewCode(code string, msg string, opts ...Option) *Error {
	e := &Error{Code: code, Msg: msg}
	return ApplyOption(e, opts...)
}
func NewSystemCode(code string, msg string, opts ...Option) *Error {
	e := &Error{Code: code, Msg: msg, System: true}
	return ApplyOption(e, opts...)
}

func NewWrap(err error, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	e := &Error{Msg: err.Error(), cause: err}
	return ApplyOption(e, opts...)
}

func NewSystemWrap(err error, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	e := &Error{Msg: err.Error(), cause: err, System: true}
	return ApplyOption(e, opts...)
}

func NewCodeWrap(code string, err error, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	e := &Error{Code: code, Msg: err.Error(), cause: err}
	return ApplyOption(e, opts...)
}

func NewSystemCodeWrap(code string, err error, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	e := &Error{Code: code, Msg: err.Error(), cause: err, System: true}
	return ApplyOption(e, opts...)
}

func Clone(err *Error, opts ...Option) *Error {
	if err == nil {
		return nil
	}
	e := &Error{
		Code:   err.Code,
		Msg:    err.Msg,
		System: err.System,
		Chain:  err.Chain,
		cause:  err.cause,
	}
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
	if errOpt.code != "" {
		err.Code = errOpt.code
	}
	if errOpt.msg != "" {
		err.Msg = errOpt.msg
	}
	if errOpt.system {
		err.System = true
	}
	if len(errOpt.chain) > 0 {
		err.Chain = append(err.Chain, errOpt.chain...)
	}
	if errOpt.cause != nil {
		err.cause = errOpt.cause
	}
	if errOpt.depth != 0 {
		err.stack = callers(3, errOpt.depth)
	}
	if len(errOpt.msgArgs) > 0 {
		err.Msg = fmt.Sprintf(err.Msg, errOpt.msgArgs...)
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
