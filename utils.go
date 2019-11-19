package goUtils

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/getsentry/raven-go"
	strip "github.com/grokify/html-strip-tags-go"
	"github.com/sirupsen/logrus"
)

func (u *Utils) Configure(options *Options) {
	u.options = options
	// Establish defaults
	if u.options.LanguageISOCode == "" {
		u.options.LanguageISOCode = "en"
	}
}

// purify : Cleanup a string from all odd characters
func (u *Utils) Purify(str string, replacer string) string {
	defaultReplacer := "-"

	// Make the string lower case
	str = strings.ToLower(str)

	// First try to replace a few more "complicated" characters...
	str = strings.Replace(str, " ", defaultReplacer, -1)
	str = strings.Replace(str, "&quot;", defaultReplacer, -1)
	str = strings.Replace(str, "&#039;", defaultReplacer, -1)
	str = strings.Replace(str, "#039;", defaultReplacer, -1)
	str = strings.Replace(str, "#39;", defaultReplacer, -1)
	str = strings.Replace(str, "&amp;", defaultReplacer, -1)

	// Replace everything that's not alpha-numeric with default replacer
	r := regexp.MustCompile(`([^a-zA-Z0-9]+)`)
	r.ReplaceAllString(str, defaultReplacer)

	// Eliminate duplicate of default replacer
	str = strings.Replace(str, strings.Repeat(defaultReplacer, 5), defaultReplacer, -1)
	str = strings.Replace(str, strings.Repeat(defaultReplacer, 4), defaultReplacer, -1)
	str = strings.Replace(str, strings.Repeat(defaultReplacer, 3), defaultReplacer, -1)
	str = strings.Replace(str, strings.Repeat(defaultReplacer, 2), defaultReplacer, -1)

	// Finally, replace the default replacer with the one passed in
	str = strings.Replace(str, defaultReplacer, u.options.PurifyReplacer, -1)
	return str
}

// UniqueInt - Remove duplicates from an array of integers
func (u *Utils) UniqueInt(input []int64) []int64 {
	r := make([]int64, 0, len(input))
	m := make(map[int64]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			r = append(r, val)
		}
	}
	return r
}

// UniqueStrings - Remove duplicates from an array of strings
func (u *Utils) UniqueStrings(input []string) []string {
	r := make([]string, 0, len(input))
	m := make(map[string]bool)
	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			r = append(r, val)
		}
	}
	return r
}

// ReportError - Report an error with Sentry (if in use) and logrus.
func (u *Utils) ReportError(location string, err error) {
	key := os.Getenv(fmt.Sprintf("%s%sSENTRY_KEY", u.options.EnvironmentVarPrefix, u.options.EnvironmentVarSeparator))
	project := os.Getenv(fmt.Sprintf("%s%sSENTRY_PROJECT", u.options.EnvironmentVarPrefix, u.options.EnvironmentVarSeparator))
	errorMessage := fmt.Sprintf("Location: %s ~ Error: %v", location, err)
	if key != "" && project != "" {
		sentryDSN := fmt.Sprintf("https://%s@%s/%s", key, u.options.SentryURL, project)
		if err := raven.SetDSN(sentryDSN); err != nil {
			logrus.Errorf("Original error: %s", errorMessage)
			logrus.Errorf("Failed to initialize Sentry! %v", err)
			return
		}
		raven.CaptureErrorAndWait(err, map[string]string{"err": errorMessage})
	}
	logrus.Error(errorMessage)
}

// LogMetrics - Log metrics into Prometheus
func (u *Utils) LogMetrics(startTime time.Time, handler string, httpCode *int) {
	timeTaken := time.Since(startTime).Seconds()
	if u.options.PrometheusRequestsVector != nil {
		u.options.PrometheusRequestsVector.WithLabelValues(fmt.Sprintf("%d", *httpCode)).Observe(timeTaken)
	}
	if u.options.PrometheusHandlersVector != nil {
		u.options.PrometheusHandlersVector.WithLabelValues(fmt.Sprintf("%d", *httpCode), handler).Observe(timeTaken)
	}
}

// SplitName - Split a full name into counter parts
func (u *Utils) SplitName(n string) (ne NameElements) {
	nameBits := strings.Split(n, " ")
	if firstName := nameBits[0]; firstName != "" {
		ne.FirstName = firstName
	}
	if len(nameBits) >= 1 {
		ne.LastName = strings.Join(nameBits[1:], " ")
	}
	return
}

// FriendlyDate - Return the passed time.Time as a friendly date (mm/dd/YYYY)
func (u *Utils) FriendlyDate(t *time.Time, includeTime bool) string {
	if t == nil {
		return "-"
	}
	dateFormat := fmt.Sprintf("%02d/%02d/%d", t.Month(), t.Day(), t.Year())
	if includeTime {
		dateFormat = fmt.Sprintf("%02d/%02d/%d %02d:%02d", t.Month(), t.Day(), t.Year(), t.Hour(), t.Minute())
	}
	return dateFormat
}

func (u *Utils) NL2BR(str string) string {
	return strings.Replace(str, "\n", "<br />", -1)
}

func (u *Utils) AddHrefs(str string) string {
	re := regexp.MustCompile(`(http|ftp|https):\/\/([\w\-_]+(?:(?:\.[\w\-_]+)+))([\w\-\.,@?^=%&amp;:/~\+#]*[\w\-\@?^=%&amp;/~\+#])?`)
	return re.ReplaceAllString(str, `<a href="$0" target="_blank">$0</a>`)
}

func (u *Utils) TokenizeStrings(normalStrings ...string) string {
	fullText := ""
	// Gather together all the strings
	for _, ns := range normalStrings {
		stripped := strip.StripTags(ns)
		fullText += " " + strings.ToLower(stripped)
	}
	// clean up any special characters
	fullTextPurified := u.Purify(fullText, " ")
	// de-duplicate the full text
	fullTextDuplicates := strings.Fields(fullTextPurified)
	fullTextDeduplicated := u.UniqueStrings(fullTextDuplicates)
	sort.Strings(fullTextDeduplicated)
	// remove stop words
	fullTextString := " " + strings.Join(fullTextDeduplicated, " ") + " "
	for _, stopWord := range StopWords[u.options.LanguageISOCode] {
		searchFor := fmt.Sprintf(" %s ", stopWord)
		fullTextString = strings.Replace(fullTextString, searchFor, " ", -1)
	}
	fullTextString = strings.Trim(fullTextString, " ")
	extraReplacements := []string{"~", "`", "!", "@", "#", "$", "%", "^", "&", "*", "(", ")", "-", "_", "=", "+", "<", ",", ">", ".", "?", "/", ":", ";", "'", "{", "[", "}", "]", "\\", "|"}
	for _, er := range extraReplacements {
		fullTextString = strings.Replace(fullTextString, er, " ", -1)
	}
	fullTextString = u.Purify(fullTextString, " ")
	fullTextString = strings.Trim(fullTextString, " ")

	// return the clean text to be inserted in the db
	return fullTextString
}

func (u *Utils) SliceHasElement(slice interface{}, element interface{}) bool {
	arrV := reflect.ValueOf(slice)
	if arrV.Kind() == reflect.Slice {
		for i := 0; i < arrV.Len(); i++ {
			if arrV.Index(i).Interface() == element {
				return true
			}
		}
	}

	return false
}

func (u *Utils) ShortenString(str string, limit int, useWords, addEllipsis bool) string {
	if str == "" {
		return ""
	}
	if len(str) < limit {
		limit = len(str)
	}
	if limit < 0 {
		limit = 0
	}
	str = strip.StripTags(str)
	if !useWords {
		return str[:limit]
	}

	strSliced := strings.Split(str, " ")
	if len(strSliced) < limit {
		limit = len(strSliced)
		addEllipsis = false
	}
	if limit < 0 {
		limit = 0
	}
	str = strings.Join(strSliced[:limit], " ")
	if addEllipsis {
		str += "..."
	}
	return str
}

func (u *Utils) GenerateSimpleUID(fromString string) string {
	fromString += "." + time.Now().Format(time.RFC3339)
	hash := md5.New()
	hash.Write([]byte(fromString))
	simpleMD5 := hex.EncodeToString(hash.Sum(nil))
	return fmt.Sprintf("%s-%s-%s-%s-%s", simpleMD5[:8], simpleMD5[8:12], simpleMD5[12:16], simpleMD5[16:20], simpleMD5[20:32])
}
