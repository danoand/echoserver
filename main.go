// echosever echoes request data back to the caller
package main

import (
	"fmt"
	"log"
	"log/syslog"
	"net/http"
	"os"

	"github.com/danoand/utils"
	"github.com/gin-gonic/gin"
)

const appName = "ECHO"

var (
	err       error
	whiteList = []string{"207.254.40.140"}

	// Papertrail logger
	pprlog *syslog.Writer
)

// appLog logs messages
func appLog(format string, v ...interface{}) {
	var cnt, msg string

	// Construct the logging line content
	cnt = fmt.Sprintf(format, v...)
	msg = fmt.Sprintf("App: %v - %v", appName, cnt)

	// Log locally
	log.Printf(msg)

	// Determine the type/level of the log message
	switch {

	// CASE: Fatal log message
	case string(cnt[:5]) == "FATAL":
		pprlog.Emerg(msg)

	// CASE: Error log message
	case string(cnt[:5]) == "ERROR":
		pprlog.Err(msg)

	// CASE: Informational log message
	case string(cnt[:4]) == "INFO":
		pprlog.Info(msg)

	// CASE: Informational log message
	case string(cnt[:5]) == "DEBUG":
		pprlog.Debug(msg)

	// DEFAULT: type/level not identified - log as informational
	default:
		pprlog.Debug(msg)
	}

	return
}

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
	fmt.Printf("INFO: %v - start logging to Papertrail\n", utils.FileLine())
	// Set up logging to Papertrail
	pprlog, err = syslog.Dial("udp", "logs.papertrailapp.com:27834", syslog.LOG_EMERG|syslog.LOG_KERN, "bvworkers")
	if err != nil {
		// error occurred dialing the remote logging service
		fmt.Printf("FATAL: %v - error occurred dialing the remote logging service. See: %v\n",
			utils.FileLine(),
			err)
		os.Exit(1)
	}

	r := gin.Default()

	// Respond to Broadvibe Worker test
	r.POST("/stubtwilio", func(c *gin.Context) {
		var (
			err    error
			rbytes []byte
		)

		// Get the request body
		rbytes, err = c.GetRawData()
		if err != nil {
			// error reading the echoserver/stubtwilio request body
			appLog("ERROR: %v - error reading the echoserver/stubtwilio request body. See: %v",
				utils.FileLine(),
				err)
			c.JSON(
				http.StatusBadRequest,
				map[string]string{"msg": "error reading the echoserver/stubtwilio request body"})
			return
		}

		appLog("INFO: %v - echoing the request body from the caller:\n%v\n",
			utils.FileLine(),
			string(rbytes))

		c.JSON(http.StatusOK, map[string]string{"msg": "logged the request body"})
	})

	// Default handler
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
