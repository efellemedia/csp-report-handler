package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type CSPReport struct {
	CSPReportDetails `json:"csp-report"`
}

type CSPReportDetails struct {
	DocumentURI        string `json:"document-uri"`
	Referrer           string `json:"referrer"`
	BlockedURI         string `json:"blocked-uri"`
	ViolatedDirective  string `json:"violated-directive"`
	EffectiveDirective string `json:"effective-directive"`
	StatusCode         int    `json:"status-code"`
	SourceFile         string `json:"source-file"`
	LineNumber         int    `json:"line-number"`
	ColumnNumber       int    `json:"column-number"`
}

const htmlTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>CSP Reports for {{.RootDomain}}</title>
<link rel="stylesheet" type="text/css" href="styles.css">
</head>
<body>
<div class="container">
    <h1>CSP Reports for {{.RootDomain}}</h1>
	<a href="/"><div class="menu">Back</div></a>
    {{range $index, $report := .Reports}}
    <div class="report">
        <button class="collapsible">Report #{{add1 $index}}: {{$report.DocumentURI}}</button>
        <div class="content">
            <p><strong>Document URI:</strong> {{$report.DocumentURI}}</p>
            <p><strong>Referrer:</strong> {{$report.Referrer}}</p>
            <p><strong>Blocked URI:</strong> {{$report.BlockedURI}}</p>
            <p><strong>Violated Directive:</strong> {{$report.ViolatedDirective}}</p>
            <p><strong>Effective Directive:</strong> {{$report.EffectiveDirective}}</p>
            <p><strong>Status Code:</strong> {{$report.StatusCode}}</p>
            <p><strong>Source File:</strong> {{$report.SourceFile}}</p>
            <p><strong>Line Number:</strong> {{$report.LineNumber}}</p>
            <p><strong>Column Number:</strong> {{$report.ColumnNumber}}</p>
        </div>
    </div>
    {{end}}
</div>
<script src="scripts.js"></script>
</body>
</html>
`

const landingPageTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>efelle creative CSP Reporting</title>
<link rel="stylesheet" type="text/css" href="styles.css">
</head>
<body>
<div class="container">
    <h1>CSP Reports for efelle creative</h1>
    <div class="search"><input type="text" id="searchInput" onkeyup="filterList()" placeholder="Search for domains."></div>
    <div class="domain-list">
        <div id="rootDomainList">
{{range $index, $rootDomain := .RootDomains}}
            <div class="domain-list-item" id="site-{{$index}}">    
                <a href="{{$rootDomain}}_csp.html">{{$rootDomain}}</a>
                <button class="delete-button" onclick="deleteSite('{{$rootDomain}}', 'site-{{$index}}')">Delete</button>
            </div>
{{end}}
        </div>
    </div>
</div>
<script src="scripts.js"></script>
</body>
</html>
`

func main() {
	http.HandleFunc("/csp-report", cspReportHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/delete-site", deleteSiteHandler)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func deleteSiteHandler(w http.ResponseWriter, r *http.Request) {
	rootDomain := r.URL.Query().Get("rootDomain")
	if rootDomain == "" {
		http.Error(w, "Root domain parameter is missing", http.StatusBadRequest)
		return
	}

	if err := deleteSiteFiles(rootDomain); err != nil {
		http.Error(w, "Error deleting site files", http.StatusInternalServerError)
		return
	}

	// Get updated list of root domains
	rootDomains := getRootDomains("static")

	// Update index.html file after deleting the site's files
	if err := updateLandingPage(rootDomains); err != nil {
		http.Error(w, "Error updating index file", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func deleteSiteFiles(rootDomain string) error {
	htmlFilePath := filepath.Join("static", rootDomain+"_csp.html")
	jsonFilePath := filepath.Join("static", rootDomain+".html")

	if err := os.Remove(htmlFilePath); err != nil {
		return err
	}

	if err := os.Remove(jsonFilePath); err != nil {
		return err
	}

	return nil
}
func updateLandingPage(rootDomains []string) error {
	tmpl, err := template.New("landing").Parse(landingPageTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse landing page template: %v", err)
	}

	landingPagePath := filepath.Join("static", "index.html")
	landingPageFile, err := os.Create(landingPagePath)
	if err != nil {
		return fmt.Errorf("failed to create landing page file: %v", err)
	}
	defer landingPageFile.Close()

	data := struct {
		RootDomains []string
	}{
		RootDomains: rootDomains,
	}

	if err := tmpl.Execute(landingPageFile, data); err != nil {
		return fmt.Errorf("failed to execute landing page template: %v", err)
	}

	return nil
}

func getRootDomains(staticDir string) []string {
	files, err := ioutil.ReadDir(staticDir)
	if err != nil {
		log.Printf("Error reading static directory: %v", err)
		return nil
	}

	var rootDomains []string
	for _, file := range files {
		if file.Mode().IsRegular() && strings.HasSuffix(file.Name(), "_csp.html") {
			rootDomain := strings.TrimSuffix(file.Name(), "_csp.html")
			rootDomains = append(rootDomains, rootDomain)
		}
	}

	return rootDomains
}

func add1(x int) int {
	return x + 1
}

func cspReportHandler(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	var report CSPReport
	err = json.Unmarshal(body, &report)
	if err != nil {
		http.Error(w, "Error parsing JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("Received CSP report: %+v\n", report)

	rootDomain, err := getRootDomain(report.CSPReportDetails.DocumentURI)
	if err != nil {
		http.Error(w, "Error extracting root domain", http.StatusInternalServerError)
		return
	}

	err = updateHTMLFile(rootDomain, &report.CSPReportDetails)
	if err != nil {
		log.Printf("Error while updating HTML file: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Update the landing page.
	rootDomains := getRootDomains("static")
	if err := updateLandingPage(rootDomains); err != nil {
		log.Printf("Error while updating landing page: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getRootDomain(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	parts := strings.Split(parsedURL.Hostname(), ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid hostname")
	}

	return strings.Join(parts[len(parts)-2:], "."), nil
}

func updateHTMLFile(rootDomain string, report *CSPReportDetails) error {
	filePath := filepath.Join("static", rootDomain+".html")
	var reports []CSPReportDetails

	if _, err := os.Stat(filePath); err == nil {
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("failed to open file '%s': %v", filePath, err)
		}
		defer file.Close()

		decoder := json.NewDecoder(file)
		if err := decoder.Decode(&reports); err != nil {
			return fmt.Errorf("failed to decode JSON from file '%s': %v", filePath, err)
		}
	}

	reports = append(reports, *report)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file '%s': %v", filePath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(reports); err != nil {
		return fmt.Errorf("failed to encode JSON to file '%s': %v", filePath, err)
	}

	tmpl, err := template.New("csp").Funcs(template.FuncMap{
		"add1": add1,
	}).Parse(htmlTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse HTML template: %v", err)
	}
	htmlFilePath := filepath.Join("static", rootDomain+"_csp.html")
	htmlFile, err := os.Create(htmlFilePath)
	if err != nil {
		return fmt.Errorf("failed to create HTML file '%s': %v", htmlFilePath, err)
	}
	defer htmlFile.Close()

	data := struct {
		RootDomain string
		Reports    []CSPReportDetails
	}{
		RootDomain: rootDomain,
		Reports:    reports,
	}

	if err := tmpl.Execute(htmlFile, data); err != nil {
		return fmt.Errorf("failed to execute HTML template: %v", err)
	}

	return nil
}
