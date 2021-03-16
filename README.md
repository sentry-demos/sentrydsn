# dsn

Written to derive DSN keys from [requests](https://golang.org/pkg/net/http/#Request) forwarded from an on prem Sentry (8.13) store endpoint /api/{projectID}/store/.

# implementation
```
import sentrydsn "github.com/sentry-demos/sentrydsn"

//some request handler

func myFunc(r *http.Request){
   
	dsn, err := sentrydsn.FromRequest(r)

	if err != nil {

		//handle err

	} 

    //check dsn length/ other logic

    myDSN := dsn.URL

    if len(myDSN) == 0{
        //handle
    }

    //return myDSN

}

```

# run tests

```go test --v```

# Limitations:
1. Currently requests sent to the legacy /api/store/ will return a DSN struct with URL as empty ""
2. Module will currently not handle forwarded requests to the sentry API: /api/0/ 
3. Module does not rewrite auth headers.





    