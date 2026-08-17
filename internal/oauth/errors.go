package oauth

import "errors"

var errInvalidRedirect = errors.New("redirect_uri must be an absolute http or https URI without a fragment")
