package gateway

import "regexp"

const emailTLDs = `com|org|net|edu|gov|mil|int|io|co|us|uk|ca|au|de|fr|es|it|nl|se|no|fi|dk|ch|at|be|pl|ru|jp|cn|in|br|mx|info|biz|me|ai|app|dev|email|mail`

var (
	emailShapeRE = regexp.MustCompile(`[A-Za-z0-9._%+\-]{2,}@[A-Za-z0-9\-]+(?:\.[A-Za-z0-9\-]+)*\.[A-Za-z]{2,}`)
	runEmailRE   = regexp.MustCompile(`^[A-Za-z0-9._%+\-]{2,}@(?:[A-Za-z0-9\-]+\.)*[A-Za-z0-9\-]{2,}\.(?i:` + emailTLDs + `)[A-Za-z]*[A-Za-z0-9]{0,2}$`)
)
