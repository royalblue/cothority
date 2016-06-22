package services

import (
	// Importing the services so they register their services to SDA
	// automatically when importing github.com/dedis/cothority/services
	_ "github.com/dedis/cothority/services/cosi"
	_ "github.com/dedis/cothority/services/status"
)
