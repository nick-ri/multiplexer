package collector

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

func (c *collector) fixedWorker(id int, paramsCh <-chan chan param) {
	log.Printf("start worker id:%d", id)
	defer log.Printf("stop worker id:%d", id)

	for {
		select {
		case ch, ok := <-paramsCh:
			if !ok {
				return
			}

			c.reader(id, ch, true)
		}
	}
}

func (c *collector) reader(id int, ch <-chan param, fixed bool) {
	log.Printf("reader id:%d, acquired", id)
	defer log.Printf("reader id:%d, released", id)

	if !fixed {
		defer c.incSpawnedCount(-1)
	}

	for prm := range ch {

		if isDoneContext(prm.ctx) {
			break
		}

		log.Printf("reader id:%d, got:%s", id, prm.url)

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
		resp.Body.Close()
		if err != nil {
			prm.errCh <- fmt.Errorf("%s :%w", prm.url, err)
			break
		}

		prm.resCh <- res{
			Url:  prm.url,
			Body: string(bts),
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
