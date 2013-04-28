package graph

import "io"
import "fmt"

func WriteDdsg(nodeCount int, edges Edgelist, w io.Writer) error {
	_, err := w.Write([]byte(fmt.Sprintf("d\n%d %d\n",
		nodeCount, len(edges))))
	if err != nil {
		return err
	}
	for _, e := range edges {
		_, err = w.Write([]byte(fmt.Sprintf("%d %d %d 0\n",
			e.n0 - 1, e.n1 - 1,  // DDSG nodes are 0-origin
			int(e.weight))))
		if err != nil {
			return err
		}
	}
	return nil
}