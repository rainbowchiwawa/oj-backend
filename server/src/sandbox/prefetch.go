package sandbox

import (
	"context"
	"io"
	"time"

	"github.com/moby/moby/client"
)

func prefetch(moby *client.Client, images []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	for _, image := range images {
		res, err := moby.ImagePull(ctx, image, client.ImagePullOptions{})
		if err != nil {
			return err
		}

		defer res.Close()
		io.Copy(io.Discard, res)
	}
	return nil
}

func WaitForPrefetch() {
	wg.Wait()
}
