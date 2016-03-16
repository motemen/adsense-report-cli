// report-adsense -no-header -metric METRIC -dimension DIMENSION -from today-7d -to today
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"code.google.com/p/goauth2/oauth"
	adsense "google.golang.org/api/adsense/v1.4"
)

type clientSecret struct {
	Web struct {
		ClientId     string `json:"client_id"`
		ClientSecret string `json:"client_secret"`
	} `json:"web"`
}

var rootDirectory string

func init() {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	rootDirectory = filepath.Join(usr.HomeDir, ".config", "adsense-report-cli")
}

func main() {
	flagSet := flag.NewFlagSet("", flag.ExitOnError)
	flagSet.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flagSet.PrintDefaults()
		fmt.Fprintln(os.Stderr, "For metrics and dimensions list, visit <https://developers.google.com/adsense/management/metrics-dimensions>")
		fmt.Fprintln(os.Stderr, "For the dates parameter, visit <https://developers.google.com/adsense/management/reporting/relative_dates>")
	}

	var (
		metric    = flagSet.String("metric", "EARNINGS", "report metric")
		dimension = flagSet.String("dimension", "DATE", "report dimension")
		from      = flagSet.String("from", "today-6d", "date range, from")
		to        = flagSet.String("to", "today", "date range, to")
		noHeader  = flagSet.Bool("no-header", false, "do not show header")
		forceAuth = flagSet.Bool("force-auth", false, "force authorization")
	)

	flagSet.Parse(os.Args[1:])

	config, err := loadOAuthConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := prepareOAuthClient(config, *forceAuth)

	adsenseService, err := adsense.New(client)
	if err != nil {
		log.Fatal(err)
	}

	reportGenerator := adsenseService.Reports.Generate(*from, *to).Metric(*metric).Dimension(*dimension).UseTimezoneReporting(true)
	res, err := reportGenerator.Do()
	if err != nil {
		log.Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, '\t', 0)

	if *noHeader == false {
		headers := []string{}
		for _, h := range res.Headers {
			header := h.Name
			if h.Currency != "" {
				header += fmt.Sprintf(" (%s)", h.Currency)
			}
			headers = append(headers, header)
		}
		fmt.Fprintln(w, strings.Join(headers, "\t"))
	}

	for _, row := range res.Rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	w.Flush()
}

func obtainToken(config *oauth.Config) *oauth.Token {
	ch := make(chan string)
	ts := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if req.URL.Path == "/favicon.ico" {
				http.Error(w, "Not Found", 404)
				return
			}

			if code := req.FormValue("code"); code != "" {
				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintln(w, "Authorized.")
				ch <- code
				return
			}

			http.Error(w, "Internal Server Error", 500)
			log.Fatalf("Could not handle request: %+v", req)
		}))
	defer ts.Close()

	config.RedirectURL = ts.URL

	authURL := config.AuthCodeURL("")

	log.Printf("Visit %s to authorize", authURL)
	exec.Command("open", authURL).Run()

	code := <-ch

	t := &oauth.Transport{Config: config}
	token, err := t.Exchange(code)
	if err != nil {
		log.Fatal(err)
	}

	return token
}

func loadOAuthConfig() (*oauth.Config, error) {
	var cs clientSecret
	err := loadJSONFromFile(filepath.Join(rootDirectory, "client_secret.json"), &cs)
	if err != nil {
		return nil, fmt.Errorf("%s; obtain one at <https://console.developers.google.com/project>", err)
	}

	config := &oauth.Config{
		ClientId:     cs.Web.ClientId,
		ClientSecret: cs.Web.ClientSecret,
		Scope:        adsense.AdsenseReadonlyScope,
		AuthURL:      "https://accounts.google.com/o/oauth2/auth",
		TokenURL:     "https://accounts.google.com/o/oauth2/token",
		AccessType:   "offline",
		TokenCache:   oauth.CacheFile(filepath.Join(rootDirectory, "auth_cache.json")),
	}

	return config, nil
}

func prepareOAuthClient(config *oauth.Config, useFresh bool) *http.Client {
	token := &oauth.Token{}

	if useFresh == false {
		err := loadJSONFromFile(filepath.Join(rootDirectory, "auth_cache.json"), token)
		if err != nil {
			useFresh = true
		}
	}

	if useFresh {
		config.ApprovalPrompt = "force"
		token = obtainToken(config)
	}

	t := &oauth.Transport{
		Token:  token,
		Config: config,
	}
	return t.Client()
}

func loadJSONFromFile(path string, v interface{}) error {
	r, err := os.Open(path)
	if err != nil {
		return err
	}

	defer r.Close()

	return json.NewDecoder(r).Decode(v)
}
