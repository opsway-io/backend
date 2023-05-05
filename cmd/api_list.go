package cmd

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labstack/echo/v4"
	"github.com/opsway-io/backend/internal/rest/controllers"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

//nolint:gochecknoglobals
var generateCmd = &cobra.Command{
	Use: "api_list",
	Run: runGenerate,
}

//nolint:gochecknoinits
func init() {
	rootCmd.AddCommand(generateCmd)
}

type Route struct {
	Method string
	Path   string
}

func runGenerate(cmd *cobra.Command, args []string) {
	e := echo.New()

	controllers.Register(
		e,
		&logrus.Entry{},
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	var routes []Route
	for _, route := range e.Routes() {
		if !isAllowedMethod(route.Method) {
			continue
		}

		if strings.Contains(route.Name, "labstack/echo") {
			continue
		}

		routes = append(routes, Route{
			Method: route.Method,
			Path:   route.Path,
		})
	}

	// sort based on path
	sort.Slice(routes, func(i, j int) bool {
		return routes[i].Path < routes[j].Path
	})

	printRoutes(routes)
}

var allowedMethods = []string{
	echo.GET,
	echo.POST,
	echo.PUT,
	echo.DELETE,
	echo.PATCH,
	echo.HEAD,
	echo.OPTIONS,
}

func isAllowedMethod(method string) bool {
	for _, m := range allowedMethods {
		if m == method {
			return true
		}
	}

	return false
}

func printRoutes(routes []Route) {
	for _, route := range routes {
		fmt.Printf("%-10s %s\n", route.Method, route.Path)
	}
}
