package collector

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func (c *collector) worker(id int, paramsCh <-chan chan param) {
	log.Printf("start worker id:%d", id)
	defer log.Printf("stop worker id:%d", id)

	for {
		select {
		case ch, ok := <-paramsCh:
			if !ok {
				return
			}

			log.Printf("worker id:%d, acquired", id)

			for prm := range ch {

				if isDoneContext(prm.ctx) {
					break
				}

				log.Printf("worker id:%d, got:%s", id, prm.url)

				req, err := http.NewRequest(http.MethodGet, prm.url, nil)
				if err != nil {
					prm.errCh <- err
					break
				}

				req = req.WithContext(prm.ctx)

				resp, err := c.client.Do(req)
				if err != nil {
					prm.errCh <- fmt.Errorf("%s :%w", prm.url, err)
					break
				}

				bts, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					prm.errCh <- fmt.Errorf("%s :%w", prm.url, err)
					break
				}

				prm.resCh <- res{
					Url:  prm.url,
					Body: string(bts),
				}

				resp.Body.Close()
			}

			log.Printf("worker id:%d, released", id)
		}
	}
}

func isDoneContext(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
