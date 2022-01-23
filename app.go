package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
)

type Cyanite struct {
	URL   string
	Token string
}

var re = regexp.MustCompile(`\n|\s+`)

func (c *Cyanite) Trim(query string) string {
	return re.ReplaceAllString(query, " ")
}

func (c *Cyanite) Query(query string, variables string) (string, error) {
	client := &http.Client{}
	query = c.Trim(query)
	variables = c.Trim(variables)
	req_body := fmt.Sprintf(`{ "query": "%s", "variables": %s }`, query, variables)

	req, err := http.NewRequest("POST", c.URL, bytes.NewReader([]byte(req_body)))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	resp.Body.Close()

	return string(body), nil
}

type CyaniteResponse struct {
	Data   interface{}    `json:"data"`
	Errors []CyaniteError `json:"errors"`
}

type CyaniteError struct {
	Message string `json:"message"`
}

type EdgeNode struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type LibraryTracksResponse struct {
	Edges []struct {
		Node EdgeNode `json:"node"`
	} `json:"edges"`
}

type FindByHash256Response struct {
	Data struct {
		LibraryTracksResponse `json:"libraryTracks"`
	} `json:"data"`
	Errors []CyaniteError `json:"errors"`
}

// Return trackId in cyanite collection
func (c *Cyanite) FindByHash256(hash256 string) (string, error) {
	query := `query LibraryTracksFilteredBySHA256Query($sha256: String!) {
				libraryTracks(filter: { sha256: $sha256 }) {
					edges {
						node {
							id
							title
						}
					}
				}
			}`
	variables := fmt.Sprintf(`{ "sha256": "%s" }`, hash256)

	resp, err := c.Query(query, variables)
	if err != nil {
		return "", err
	}

	response := &FindByHash256Response{}
	err = json.Unmarshal([]byte(resp), response)
	if len(response.Errors) > 0 {
		return "", errors.New(fmt.Sprintf("GraphQL error: %s", response.Errors[0].Message))
	}

	if len(response.Data.LibraryTracksResponse.Edges) == 0 {
		return "", errors.New("No track found by hash")
	}

	return response.Data.LibraryTracksResponse.Edges[0].Node.ID, nil
}

type SimilarTracks struct {
	Edges []struct {
		Node EdgeNode `json:"node"`
	} `json:"edges"`
}

type LibraryTrack struct {
	ID            string        `json:"id"`
	SimilarTracks SimilarTracks `json:"similarTracks"`
}

type FindSimilarResponse struct {
	Data struct {
		LibraryTrack `json:"libraryTrack"`
	} `json:"data"`
	Errors []CyaniteError `json:"errors"`
}

func (c *Cyanite) FindSimilar(trackID string) ([]string, error) {
	query := `query SimilarTracksQuery($trackId: ID!) {
		libraryTrack(id: $trackId) {
		  ... on LibraryTrack {
			id
			similarTracks(target: {spotify: {}}) {
			  ... on SimilarTracksConnection {
				edges {
				  node {
					... on SpotifyTrack {
					  id
					}
				  }
				}
			  }
			}
		  }
		}
	  }
	  `
	variables := fmt.Sprintf(`{ "trackId": "%s" }`, trackID)

	resp, err := c.Query(query, variables)
	if err != nil {
		return nil, err
	}

	response := &FindSimilarResponse{}
	err = json.Unmarshal([]byte(resp), response)

	if len(response.Errors) > 0 {
		return nil, errors.New(fmt.Sprintf("GraphQL error: %s", response.Errors[0].Message))
	}

	result := []string{}

	for _, node := range response.Data.SimilarTracks.Edges {
		result = append(result, node.Node.ID)
	}

	return result, nil
}

func main() {
	cyanite := &Cyanite{
		URL:   "https://api.cyanite.ai/graphql",
		Token: "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXBlIjoiSW50ZWdyYXRpb25BY2Nlc3NUb2tlbiIsInZlcnNpb24iOiIxLjAiLCJpbnRlZ3JhdGlvbklkIjoyMTUsInVzZXJJZCI6NTgwMywiYWNjZXNzVG9rZW5TZWNyZXQiOiIyZDg2MzBjYTVmNzRiZTgwOWI1MmRlNmEyNDcxMTdkZjZkYWQ3ZmE2ZDZiMmI3ODZkYzJkMjUxNTg0YjIzM2QwIiwiaWF0IjoxNjQyOTM3Mjg5fQ.BE3xidn1paQRIm7LQmb46FKKknF9PA2zRYGph0Yyxos",
	}
	trackId, err := cyanite.FindByHash256("c5307921c306a78aa70edeb2d1976af0b1ea1e40c616abfa68b6cdc39484964e")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(trackId)
	similar, err := cyanite.FindSimilar(trackId)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(similar)
}
