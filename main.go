package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	apiBase = "https://bsky.social/xrpc"
)

// API request/response types

type createSessionRequest struct {
	Identifier string `json:"identifier"`
	Password   string `json:"password"`
}

type createSessionResponse struct {
	AccessJwt string `json:"accessJwt"`
	DID       string `json:"did"`
	Handle    string `json:"handle"`
}

type profileView struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
}

type getFollowsResponse struct {
	Follows []profileView `json:"follows"`
	Cursor  string        `json:"cursor"`
}

type listRecord struct {
	Type        string `json:"$type"`
	Purpose     string `json:"purpose"`
	Name        string `json:"name"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
}

type listItemRecord struct {
	Type      string `json:"$type"`
	Subject   string `json:"subject"`
	List      string `json:"list"`
	CreatedAt string `json:"createdAt"`
}

type createRecordRequest struct {
	Repo       string      `json:"repo"`
	Collection string      `json:"collection"`
	Record     interface{} `json:"record"`
}

type createRecordResponse struct {
	URI string `json:"uri"`
	CID string `json:"cid"`
}

type apiError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// Client for Bluesky API
type Client struct {
	httpClient *http.Client
	accessJwt  string
	did        string
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) doRequest(method, endpoint string, body interface{}, authenticated bool) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, apiBase+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if authenticated && c.accessJwt != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessJwt)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr apiError
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiErr.Message)
		}
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *Client) CreateSession(handle, password string) error {
	reqBody := createSessionRequest{
		Identifier: handle,
		Password:   password,
	}

	respBody, err := c.doRequest("POST", "/com.atproto.server.createSession", reqBody, false)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	var resp createSessionResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return fmt.Errorf("parse session response: %w", err)
	}

	c.accessJwt = resp.AccessJwt
	c.did = resp.DID
	return nil
}

func (c *Client) GetFollows(handle string) ([]profileView, error) {
	var allFollows []profileView
	cursor := ""

	for {
		endpoint := fmt.Sprintf("/app.bsky.graph.getFollows?actor=%s&limit=100", url.QueryEscape(handle))
		if cursor != "" {
			endpoint += "&cursor=" + url.QueryEscape(cursor)
		}

		respBody, err := c.doRequest("GET", endpoint, nil, true)
		if err != nil {
			return nil, fmt.Errorf("get follows: %w", err)
		}

		var resp getFollowsResponse
		if err := json.Unmarshal(respBody, &resp); err != nil {
			return nil, fmt.Errorf("parse follows response: %w", err)
		}

		allFollows = append(allFollows, resp.Follows...)
		fmt.Printf("\r  Fetched %d follows...", len(allFollows))

		if resp.Cursor == "" {
			break
		}
		cursor = resp.Cursor
	}
	fmt.Println()

	return allFollows, nil
}

func (c *Client) CreateList(name string) (string, error) {
	record := listRecord{
		Type:        "app.bsky.graph.list",
		Purpose:     "app.bsky.graph.defs#curatelist",
		Name:        name,
		Description: "",
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}

	reqBody := createRecordRequest{
		Repo:       c.did,
		Collection: "app.bsky.graph.list",
		Record:     record,
	}

	respBody, err := c.doRequest("POST", "/com.atproto.repo.createRecord", reqBody, true)
	if err != nil {
		return "", fmt.Errorf("create list: %w", err)
	}

	var resp createRecordResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("parse create list response: %w", err)
	}

	return resp.URI, nil
}

func (c *Client) AddListItem(listURI, subjectDID string) error {
	record := listItemRecord{
		Type:      "app.bsky.graph.listitem",
		Subject:   subjectDID,
		List:      listURI,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	reqBody := createRecordRequest{
		Repo:       c.did,
		Collection: "app.bsky.graph.listitem",
		Record:     record,
	}

	_, err := c.doRequest("POST", "/com.atproto.repo.createRecord", reqBody, true)
	if err != nil {
		return fmt.Errorf("add list item: %w", err)
	}

	return nil
}

func usage() {
	fmt.Fprintf(os.Stderr, `bsky-spy - Create a Bluesky list from someone's follows

Usage:
  bsky-spy --name "List Name" <handle>

Arguments:
  handle         Bluesky handle to copy follows from (e.g., user.bsky.social)

Flags:
  --name, -n     Custom name for the list (required)
  --help, -h     Show this help message

Environment variables:
  BSKY_HANDLE    Your Bluesky handle
  BSKY_APP_KEY   Your app password (Settings > App Passwords)

Example:
  BSKY_HANDLE=me.bsky.social BSKY_APP_KEY=xxxx bsky-spy --name "Tech Folks" techperson.bsky.social
`)
	os.Exit(1)
}

func main() {
	// Parse arguments
	args := os.Args[1:]
	var listName, targetHandle string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "--help" || arg == "-h":
			usage()
		case arg == "--name" || arg == "-n":
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --name requires a value")
				os.Exit(1)
			}
			i++
			listName = args[i]
		case strings.HasPrefix(arg, "--name="):
			listName = strings.TrimPrefix(arg, "--name=")
		case strings.HasPrefix(arg, "-n="):
			listName = strings.TrimPrefix(arg, "-n=")
		case !strings.HasPrefix(arg, "-"):
			if targetHandle == "" {
				targetHandle = arg
			} else {
				fmt.Fprintf(os.Stderr, "Error: unexpected argument: %s\n", arg)
				os.Exit(1)
			}
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown flag: %s\n", arg)
			os.Exit(1)
		}
	}

	// Validate required arguments
	if listName == "" {
		fmt.Fprintln(os.Stderr, "Error: --name flag is required")
		usage()
	}
	if targetHandle == "" {
		fmt.Fprintln(os.Stderr, "Error: handle argument is required")
		usage()
	}

	// Get credentials from environment
	handle := os.Getenv("BSKY_HANDLE")
	appKey := os.Getenv("BSKY_APP_KEY")

	if handle == "" || appKey == "" {
		fmt.Fprintln(os.Stderr, "Error: BSKY_HANDLE and BSKY_APP_KEY environment variables are required")
		os.Exit(1)
	}

	client := NewClient()

	// Authenticate
	fmt.Println("Authenticating...")
	if err := client.CreateSession(handle, appKey); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Fetch follows
	fmt.Printf("Fetching follows for %s...\n", targetHandle)
	follows, err := client.GetFollows(targetHandle)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if len(follows) == 0 {
		fmt.Println("No follows found for this user.")
		os.Exit(0)
	}

	fmt.Printf("Found %d follows\n", len(follows))

	// Create list
	fmt.Printf("Creating list \"%s\"...\n", listName)
	listURI, err := client.CreateList(listName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Add members
	fmt.Println("Adding members to list...")
	for i, follow := range follows {
		if err := client.AddListItem(listURI, follow.DID); err != nil {
			fmt.Fprintf(os.Stderr, "\nWarning: failed to add %s: %v\n", follow.Handle, err)
			continue
		}
		fmt.Printf("\r  Added %d/%d members...", i+1, len(follows))

		// Rate limiting delay
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println()

	fmt.Printf("Done! List \"%s\" created with %d members.\n", listName, len(follows))
	fmt.Printf("View at: https://bsky.app/profile/%s/lists\n", handle)
}
