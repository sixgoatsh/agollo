package remote

import "fmt"

func agolloKey(appID, endpoint string) string {
	return fmt.Sprintf("%s-%s", appID, endpoint)
}
