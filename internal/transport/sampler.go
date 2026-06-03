package transport

import _ "embed"

//go:generate bash sampler/assemble.sh

//go:embed sampler.sh
var remoteSampler string
