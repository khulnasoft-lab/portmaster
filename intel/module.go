package intel

import (
	_ "github.com/khulnasoft-lab/portmaster/intel/customlists"
	"github.com/safing/portbase/modules"
)

// Module of this package. Export needed for testing of the endpoints package.
var Module *modules.Module

func init() {
	Module = modules.Register("intel", nil, nil, nil, "geoip", "filterlists", "customlists")
}
