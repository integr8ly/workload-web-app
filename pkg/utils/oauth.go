package utils

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
	"golang.org/x/net/publicsuffix"
)

const testingIDP = "testing-idp"

type goqueryNode interface {
	Find(selector string) *goquery.Selection
}

// Create an HTTP Client with stored cookies and allowed redirects
func NewOAuthHTTPClient() (*http.Client, error) {

	// create the http client with a cookie jar
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, fmt.Errorf("failed to create new cookie jar: %v", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{},
	}

	c := &http.Client{
		Jar:           jar,
		Transport:     transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return nil },
	}

	return c, nil
}

// Login a user through the oauth proxy for a product
func AuthenticateClientThroughProxyOAuth(client *http.Client, host string, username string, password string) error {

	// Start the authentication
	u := fmt.Sprintf("%s/oauth/start", host)
	response, err := client.Get(u)
	if err != nil {
		return fmt.Errorf("failed to open %s: %s", u, err)
	}
	if response.StatusCode != 200 {
		return errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}

	// Select the testing IDP
	document, err := parseHtmlResponse(response)
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// find the link to the testing IDP
	link, err := findElement(document, fmt.Sprintf("a:contains('%s')", testingIDP))
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// get the url from the link
	href, err := getAttribute(link, "href")
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	u, err = resolveRelativeURL(response, href)
	if err != nil {
		return err
	}

	// open the IDP link
	response, err = client.Get(u)
	if err != nil {
		return fmt.Errorf("failed to request %s: %s", u, err)
	}
	if response.StatusCode != 200 {
		return errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}

	// Submit the username and password
	document, err = parseHtmlResponse(response)
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// find the form for the login
	form, err := findElement(document, "#kc-form-login")
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// retrieve the action of the form
	action, err := getAttribute(form, "action")
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	u, err = resolveRelativeURL(response, action)
	if err != nil {
		return err
	}

	// submit the form with the username and password
	v := url.Values{"username": []string{username}, "password": []string{password}}
	response, err = client.PostForm(u, v)
	if err != nil {
		return fmt.Errorf("failed to request %s: %s", u, err)
	}
	if response.StatusCode != 200 {
		return errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}

	// If sucessfully redirect to the start point we are authenticated
	err = verifyRedirectURL(response.Request.URL.String(), host)
	if err == nil {
		return nil
	}

	// On first login the user is presented with an approval form. We have to submit
	// the form along with the scopes that we want to grant.
	document, err = parseHtmlResponse(response)
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// No form found: no further approval required, we are authenticated
	// at this point
	form, err = findElement(document, "form")
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// retrieve the action of the form
	action, err = getAttribute(form, "action")
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// form submit url
	u, err = resolveRelativeURL(response, action)
	if err != nil {
		return err
	}

	then, err := findElementAndGetAttribute(form, "input[name='then']", "value")
	if err != nil {
		return errorWithResponseDump(response, err)
	}
	csrf, err := findElementAndGetAttribute(form, "input[name='csrf']", "value")
	if err != nil {
		return errorWithResponseDump(response, err)
	}
	clientId, err := findElementAndGetAttribute(form, "input[name='client_id']", "value")
	if err != nil {
		return errorWithResponseDump(response, err)
	}
	userName, err := findElementAndGetAttribute(form, "input[name='user_name']", "value")
	if err != nil {
		return errorWithResponseDump(response, err)
	}
	redirectUrl, err := findElementAndGetAttribute(form, "input[name='redirect_uri']", "value")
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	// submit the form to accept the permissions
	v = url.Values{
		"then":         []string{then},
		"csrf":         []string{csrf},
		"client_id":    []string{clientId},
		"user_name":    []string{userName},
		"redirect_uri": []string{redirectUrl},
		"scope":        []string{"user:info", "user:check-access"},
		"approve":      []string{"Allow+selected+permissions"},
	}
	response, err = client.PostForm(u, v)
	if err != nil {
		return fmt.Errorf("failed to request %s: %s", u, err)
	}
	if response.StatusCode != 200 {
		return errorWithResponseDump(response, fmt.Errorf("the request to %s failed with code %d", u, response.StatusCode))
	}

	// verify that we have been ridirect to the correct host
	err = verifyRedirectURL(response.Request.URL.String(), host)
	if err != nil {
		return errorWithResponseDump(response, err)
	}

	return nil
}

func verifyRedirectURL(redirectUrl, host string) error {
	if !strings.HasPrefix(redirectUrl, host) {
		return fmt.Errorf("redirect host does not match product host: %v / %v", redirectUrl, host)
	}
	return nil
}

func dumpResponse(r *http.Response) string {
	msg := "> Request\n"
	reqOut, err := httputil.DumpRequestOut(r.Request, false)
	if err != nil {
		msg += fmt.Sprintf("failed to dump the request: %s", err)
	} else {
		msg += string(reqOut)
	}
	msg += "\n"

	msg += "< Response\n"
	for name, values := range r.Header {
		// Loop over all values for the name.
		for _, value := range values {
			msg += name + " " + value + "\n"
		}
	}

	document, err := parseHtmlResponse(r)
	if err != nil {
		msg += "parse html response failed"
	} else {
		selection := document.Find("body")
		if selection.Length() == 1 {
			var b bytes.Buffer
			err = html.Render(&b, selection.Nodes[0])
			msg += b.String()
		}
	}

	msg += "\n"

	return msg
}

func errorWithResponseDump(r *http.Response, err error) error {
	return fmt.Errorf("%s\n\n%s", err, dumpResponse(r))
}

func parseHtmlResponse(r *http.Response) (*goquery.Document, error) {
	// Clone the body while reading it so that in case of errors
	// we can dump the response with the body
	var clone bytes.Buffer
	body := io.TeeReader(r.Body, &clone)
	r.Body = ioutil.NopCloser(&clone)

	d, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return nil, fmt.Errorf("failed to create the document: %s", err)
	}

	// <noscript> bug workaround
	// https://github.com/PuerkitoBio/goquery/issues/139#issuecomment-517526070
	d.Find("noscript").Each(func(i int, s *goquery.Selection) {
		s.SetHtml(s.Text())
	})

	return d, nil
}

func resolveRelativeURL(r *http.Response, relativeURL string) (string, error) {
	u, err := url.Parse(relativeURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse the url %s: %s", relativeURL, err)
	}

	u = r.Request.URL.ResolveReference(u)

	return u.String(), nil
}

func findElement(node goqueryNode, selector string) (*goquery.Selection, error) {
	e := node.Find(selector)
	if e.Length() == 0 {
		return nil, fmt.Errorf("failed to find an element matching the selector %s", selector)
	}
	if e.Length() > 1 {
		return nil, fmt.Errorf("multiple element founded matching the selector %s", selector)
	}

	return e, nil
}

func getAttribute(element *goquery.Selection, name string) (string, error) {
	v, ok := element.Attr(name)
	if !ok {
		e, err := element.Html()
		if err != nil {
			e = fmt.Sprintf("failed to get the html content: %s", err)
		}

		return "", fmt.Errorf("the element '%s' doesn't have the %s attribute", e, name)
	}
	return v, nil
}

func findElementAndGetAttribute(node goqueryNode, elementSelector, attributeName string) (string, error) {

	element, err := findElement(node, elementSelector)
	if err != nil {
		return "", err
	}

	return getAttribute(element, attributeName)
}
