package admin

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/qor/qor"
	"github.com/qor/roles"
)

// RouteConfig config for admin routes
type RouteConfig struct {
	Resource       *Resource
	Permissioner   HasPermissioner
	PermissionMode roles.PermissionMode
	Values         map[interface{}]interface{}
}

type requestHandler func(c *Context)

type routeHandler struct {
	Path   string
	Handle requestHandler
	Config RouteConfig
}

func newRouteHandler(path string, handle requestHandler, configs ...RouteConfig) routeHandler {
	handler := routeHandler{
		Path:   "/" + strings.TrimPrefix(path, "/"),
		Handle: handle,
	}

	for _, config := range configs {
		handler.Config = config
	}

	if handler.Config.Permissioner == nil && handler.Config.Resource != nil {
		handler.Config.Permissioner = handler.Config.Resource
	}
	return handler
}

var emptyPermissionMode roles.PermissionMode

func (handler routeHandler) HasPermission(context *qor.Context) bool {
	if handler.Config.Permissioner == nil || handler.Config.PermissionMode == emptyPermissionMode {
		return true
	}
	return handler.Config.Permissioner.HasPermission(handler.Config.PermissionMode, context)
}

func isAlpha(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func isAlnum(ch byte) bool {
	return isAlpha(ch) || isDigit(ch)
}

func matchPart(b byte) func(byte) bool {
	return func(c byte) bool {
		return c != b && c != '/'
	}
}

func match(s string, f func(byte) bool, i int) (matched string, next byte, j int) {
	j = i
	for j < len(s) && f(s[j]) {
		j++
	}
	if j < len(s) {
		next = s[j]
	}
	return s[i:j], next, j
}

// mostly copied from pat https://github.com/bmizerany/pat
func (handler routeHandler) try(path string) (url.Values, bool) {
	p := make(url.Values)
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(handler.Path):
			if handler.Path != "/" && len(handler.Path) > 0 && handler.Path[len(handler.Path)-1] == '/' {
				return p, true
			}
			return nil, false
		case handler.Path[j] == ':':
			var name, val string
			var nextc byte

			name, nextc, j = match(handler.Path, isAlnum, j+1)
			val, _, i = match(path, matchPart(nextc), i)

			if (j < len(handler.Path)) && handler.Path[j] == '[' {
				var index int
				if idx := strings.Index(handler.Path[j:], "]/"); idx > 0 {
					index = idx
				} else if handler.Path[len(handler.Path)-1] == ']' {
					index = len(handler.Path) - j - 1
				}

				if index > 0 {
					match := strings.TrimSuffix(strings.TrimPrefix(handler.Path[j:j+index+1], "["), "]")
					if reg, err := regexp.Compile("^" + match + "$"); err == nil && reg.MatchString(val) {
						j = j + index + 1
					} else {
						return nil, false
					}
				}
			}

			p.Add(":"+name, val)
		case path[i] == handler.Path[j]:
			i++
			j++
		default:
			return nil, false
		}
	}

	if j != len(handler.Path) {
		return nil, false
	}
	return p, true
}
