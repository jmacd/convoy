package boards;

import "log"
import "io/ioutil"
import "net/http"

func ReadUrl(host, uri, query string) ([]byte, error) {
	url := "http://" + host + uri + query
	log.Println("Trying", url)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}