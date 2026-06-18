/*
*
@Time : 2026/03/08 12:28
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/
package hikisapi

import (
	"fmt"
	"io"
	"testing"

	digest "github.com/xinsnake/go-http-digest-auth-client"
)

func TestDeviceInfoAPI(t *testing.T) {

	url := "http://192.168.1.105/ISAPI/System/deviceinfo"

	dr := digest.NewRequest("admin", "1234abcd", "GET", url, "")

	resp, err := dr.Execute()

	if err != nil {
		t.Fatal(err)
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	fmt.Println("Status:", resp.Status)
	fmt.Println(string(body))
}
