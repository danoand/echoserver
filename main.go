// echosever echoes request data back to the caller
package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/danoand/utils"
	"github.com/gin-gonic/gin"
)

var (
	whiteList = []string{"207.254.40.140"}
)

// responseObject models the data to be sent back to the caller
type responseObject struct {
	InURI         string                 `json:"inuri"`
	InHeaders     map[string][]string    `json:"inheaders"`
	InQueryString string                 `json:"inquerystring"`
	InBody        map[string]interface{} `json:"inbody"`
}

// hlprIsNotIn determines if a string is NOT one of a number of strings in a set
func hlprIsNotIn(tst string, set ...string) (rbool bool) {
	rbool = true

	// Iterate through the set
	for _, val := range set {
		if tst == val {
			// Found the test value in the lookup set
			rbool = false
		}
	}

	return
}

func main() {
	r := gin.Default()

	// Define the handler
	r.NoRoute(func(c *gin.Context) {
		var (
			err    error
			resp   responseObject
			reqStr string
			bbody  []byte
		)

		ipadr := c.ClientIP()
		if !hlprIsNotIn(ipadr, whiteList...) {
			// ip address is not valid
			c.JSON(http.StatusNotImplemented, "not implemented")
			return
		}

		// Dump the request to the log
		reqStr, _, err = utils.DumpRequest(c.Request)
		if err != nil {
			// error occurred dumping the inbound request
			log.Printf("ERROR: %v - error occurred dumping the inbound request\n", utils.FileLine())
		}
		if err == nil {
			// dump the inbound request
			log.Printf("\nINFO: %v - inbound request from %v\n%v\n",
				utils.FileLine(),
				c.ClientIP(),
				reqStr)
		}

		// Echo selected request data elements
		// ** URI Data
		resp.InURI = fmt.Sprintf("%v%v", c.Request.Host, c.Request.URL.Path)

		//** Query Parameter Data
		resp.InQueryString = c.Request.URL.RawQuery

		//** Header Data
		resp.InHeaders = c.Request.Header

		//** Parsed Body (assume json)
		bbody, err = c.GetRawData()
		if err != nil {
			// error occurred reading the body data
			log.Printf("ERROR: %v - error occurred reading the body data\n", utils.FileLine())
			resp.InBody = map[string]interface{}{"msg": "body can't be processed"}
			c.JSON(http.StatusOK, resp)
			return
		}

		// Read the request body; assign to the response
		tMap := make(map[string]interface{})
		err = utils.FromJSONBytes(bbody, &tMap)
		if err != nil {
			// error occurred parsing the body data as json
			log.Printf("ERROR: %v - error occurred parsing the body data as json\n", utils.FileLine())
			resp.InBody = map[string]interface{}{"msg": "error occurred parsing the body data as json"}
			c.JSON(http.StatusOK, resp)
			return
		}

		// ** return value
		resp.InBody = tMap
		c.JSON(http.StatusOK, resp)
	})

	// Start the server
	log.Printf("INFO: starting the webserver on port: 8999\n")
	r.Run(":8999") // listen and serve on 0.0.0.0:8999
}
