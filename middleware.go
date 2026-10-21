package validate

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/layer8co/caddy-validate-github/matcher"

	"go.uber.org/zap"
)

var (
	_ caddy.Module                = (*Middleware)(nil)
	_ caddy.Provisioner           = (*Middleware)(nil)
	_ caddy.Validator             = (*Middleware)(nil)
	_ caddyfile.Unmarshaler       = (*Middleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*Middleware)(nil)
)

func init() {
	caddy.RegisterModule(Middleware{})
	httpcaddyfile.RegisterHandlerDirective("validate_github_webhook", parseCaddyfileHandler)
}

type Middleware struct {
	Secret string `json:"secret,omitempty"`

	// Match contains JSON paths and their expected string values.
	// All conditions must match.
	Match map[string]string `json:"match,omitempty"`

	logger *zap.SugaredLogger
	secret []byte
}

// CaddyModule returns the Caddy module information.
func (Middleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.validate_github_webhook",
		New: func() caddy.Module { return new(Middleware) },
	}
}

// Provision implements the caddy.Provisioner interface.
func (m *Middleware) Provision(ctx caddy.Context) error {
	if m.logger == nil {
		m.logger = ctx.Logger(m).Sugar()
	}

	if m.Match == nil {
		m.Match = make(map[string]string)
	}

	return nil
}

// ServeHTTP implements the caddy.Handler interface.
func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request,
	next caddyhttp.Handler) error {
	b, err := io.ReadAll(r.Body)
	if err != nil {
		m.logger.Errorf("reading request body: %v", err)
		http.Error(w, "invalid signature", http.StatusForbidden)
		return nil
	}
	_ = r.Body.Close()

	if len(b) == 0 {
		m.logger.Debug("cannot validate an empty request body")
		http.Error(w, "invalid signature", http.StatusForbidden)
		return nil
	}

	s := strings.TrimPrefix(r.Header.Get("X-Hub-Signature-256"), "sha256=")
	if s == "" {
		m.logger.Debug("missing X-Hub-Signature-256 header in request")
		http.Error(w, "invalid signature", http.StatusForbidden)
		return nil
	}

	sig, err := hex.DecodeString(s)
	if err != nil {
		m.logger.Debugf("error hex-decoding signature '%s': %v", s, err)
		http.Error(w, "invalid signature", http.StatusForbidden)
		return nil
	}

	mac := hmac.New(sha256.New, m.secret)
	mac.Write(b)
	if sum := mac.Sum(nil); !hmac.Equal(sum, sig) { // constant time comparison
		m.logger.Debugf("signature: expected '%x'; received '%s'", sum, s)
		http.Error(w, "invalid signature", http.StatusForbidden)
		return nil
	}

	// Now, Checking for the match conditions (If any)
	if len(m.Match) != 0 {
		yes, err := matcher.MatchesPayload(b, m.Match)
		if err != nil {
			m.logger.Debugf("Error when checking the match conditions: %s", err)
			return nil
		}

		if !yes {
			m.logger.Debugf("The match condition aren't met with the request from %s", r.RemoteAddr)
			return nil
		}
	}

	m.logger.Infof("successful github webhook invocation from %s", r.RemoteAddr)
	r.Body = io.NopCloser(bytes.NewBuffer(b))

	return next.ServeHTTP(w, r)
}

func (m *Middleware) Validate() error {
	if m.Secret == "" {
		return fmt.Errorf("empty secret")
	}

	m.secret = []byte(m.Secret)

	return nil
}

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
// First Syntax:
//
//	validate_github_webhook <secret>
//
// Second Syntax:
// validate_github_webhook {
// secret <secret>
//
//	match {
//	   workflow_run.status completed
//	}
func (m *Middleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		// The first syntax
		if d.NextArg() {
			m.Secret = d.Val()
			if d.NextArg() {
				return d.ArgErr()
			}
		}

		// The modern syntax
		for level := d.Nesting(); d.NextBlock(level); {
			switch d.Val() {
			case "secret":
				// Just load the secret then
				if !d.NextArg() {
					return d.ArgErr()
				}

				m.Secret = d.Val()

				if d.NextArg() {
					return d.ArgErr()
				}

			case "match":
				for nesting := d.Nesting(); d.NextBlock(nesting); {
					// Example of the syntax:
					// workflow_run.status completed
					//
					// "workflow_run.status" is the path and "completed" is the value
					path := d.Val()

					if !d.NextArg() {
						return d.ArgErr()
					}

					value := d.Val()

					if d.NextArg() {
						return d.ArgErr()
					}

					if _, exists := m.Match[path]; exists {
						return d.Errf("there is a duplicate match condition %q", path)
					}

					m.Match[path] = value
				}

			}

		}

	}

	return nil
}

func parseCaddyfileHandler(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler,
	error) {
	m := new(Middleware)
	if err := m.UnmarshalCaddyfile(h.Dispenser); err != nil {
		return nil, err
	}

	return m, nil
}
