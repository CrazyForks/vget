How to sign URLs for BunnyCDN Token Authentication
August 21, 2020
BunnyCDN provides a powerful token authentication system to strictly control who, where and for how long can access your content.

This guide contains the documentation on how to enable, configure and generate the tokens to securely access your content. If you are looking for our older, but much more simple token authentication system, please check older token authentication guide instead. Both can be used interchangeably and our system will automatically detect and validate both token types.

What is token authentication?
First of all, what is token authentication? In short, if enabled, token authentication will block all requests to your URLs unless a valid token is passed with the request. The token is then compared on our server and we check if both values match. If the hash we generate matches the one sent by the request, then the request is accepted, otherwise, a 403 page is returned.

The token can then either be put in as a query parameter or used as part of the URL path. The path version is useful for situations such as video delivery. Here are two examples of signed URLs:

https://test.b-cdn.net/my-partial/url/video.mp4?token=aiDhHXZGHidbDZZqIY14z1eHhhbZvMlzh_7u9cEIdfI&token_path=%2Fmy-partial%2Furl%2F&expires=1598024587
https://test.b-cdn.net/bcdn_token=aiDhHXZGHidbDZZqIY14z1eHhhbZvMlzh_7u9cEIdfI&expires=1598024587&token_path=%2Fmy-partial%2Furl%2F/my-partial/url/video.mp4

How to sign an URL - Part 1: Parameters
This section contains instructions on how to generate and format the unique tokens and use those to sign an URL. We also provide code examples and helper functions for popular programming languages that allow you to sign an URL with a simple function call.

The signing process consists of the following parameters that need to be added to the URL:

token (required)
expires (required)
token_path (optional)
token_countries (optional)
token_countries_blocked (optional)
token
The token parameter is the main key in signing the URL. It represents a Base64 encoded SHA256 hash based on the URL, expiration time and other parameters. We will show how to generate the token in the next section.

expires
The expires parameter is the second main parameter that must always be included. It allows you to control exactly for how long the signed URL is valid. It is a UNIX timestamp based time representation. Any request after this timestamp will be rejected.

For example, a token expiring on 08/21/2020 @ 3:43 pm (UTC) would use:

&expires=1598024587
token_path (optional)
By default, the full request URL is used for signing a request. The token_path parameter allows you to specify a partial path that will be used instead of the full URL.

Let's use this URL for an example:

https://test.b-cdn.net/my-partial/url/playlist.m3u8
By default, the full path would need to be used when generating the token, and only this specific path would be allowed to be accessed with the specific token:

/my-partial/url/playlist.m3u8
However, we by passing the token_path for:

&token_path=%2Fmy-partial%2Furl%2F
That would create a token that has access to any file within that path, for example:

https://test.b-cdn.net/my-partial/url/playlist.m3u8
https://test.b-cdn.net/my-partial/url/file1.ts
https://test.b-cdn.net/my-partial/url/file2.ts
https://test.b-cdn.net/my-partial/url/file3.ts
Would all be covered by the token. This is useful for video delivery.

token_countries (optional)
The token_countries allows you to specify a list of countries that will have access to the URL. Any request outside of these countries will be rejected. It is a comma-separated list, for example:

&token_countries=SI,GB
token_countries_blocked (optional)
The token_countries_blocked is similar to token_countries. It allows you to specify a list of countries that will not have access to the URL. Any request from one of the listed countries will be rejected. It is a comma-separated list, for example:

&token_countries_blocked=SI,GB

How to sign a URL - Part 2: Generating the token
Now that we understand all the parameters, we can proceed to actually generate the token. The token is a Base64 encoded raw SHA256 hash based on the signed URL, expiration time, and any extra parameters. To generate the token, you can use the following algorithm:

Base64Encode(
SHA256_RAW(token_security_key + signed_url + expiration + (optional)remote_ip + (optional)encoded_query_parameters)
)
When generating the token, all of the URL query parameters must also be appended at the end of the hashable string in an ascending order except for the "token" and "expires" parameter which should not be included. These must NOT be URL encoded, but they need to be in the form-encoded POST format without the starting "?" character, for example:

"param1=something&param2=something&param3=something"
An example hashable base for a SHA256 token would then be:

security-key/my-directory/12345192.168.1.1token*countries=SI,GB&width=500&token_path=/my-directory/
To properly format the token you have to then replace the following characters in the resulting Base64 string: '\n' with '', '+' with '-', '/' with '*' and '=' with ''.

How to sign an URL - Part 3: Putting it all together
Once we have the token and all the parameters, it's time to put it all together. There are currently two ways of signing the URL.

Query Parameter Based Tokens
The easiest way is to append the token to the request using query parameters, for example:

https://test.b-cdn.net/my-partial/url/video.mp4?token=aiDhHXZGHidbDZZqIY14z1eHhhbZvMlzh_7u9cEIdfI&token_path=%2Fmy-partial%2Furl%2F&expires=1598024587
URL Path Based Tokens
The second option are the URL path based tokens that are useful for video delivery because any sub-request to the same directory of the URL will already have the token included.

https://test.b-cdn.net/bcdn_token=aiDhHXZGHidbDZZqIY14z1eHhhbZvMlzh_7u9cEIdfI&expires=1598024587&token_path=%2Fmy-partial%2Furl%2F/my-partial/url/video.mp4
Note that when generating the token, the token_path parameter does not need to include the signature path.

If it all went well, you should now have secure URLs that are only accessible through the signed URLs. If you experience any difficulties generating the tokens, please reach out to support@bunny.net, and we'll be happy to help with the implementation.

nodejs exmaple:

```javascript
var crypto = require("crypto"),
  securityKey = "229248f0-f007-4bf9-ba1f-bbf1b4ad9d40",
  path = "/300kb.jpg";

// Set the time of expiry to one hour from now
var expires = Math.round(Date.now() / 1000) + 3600;

var hashableBase = securityKey + path + expires;

// If using IP validation
// hashableBase += "146.14.19.7";

// Generate and encode the token
var md5String = crypto.createHash("md5").update(hashableBase).digest("binary");
var token = new Buffer(md5String, "binary").toString("base64");
token = token.replace(/\+/g, "-").replace(/\//g, "_").replace(/\=/g, "");

// Generate the URL
var url =
  "https://token-tester.b-cdn.net" +
  path +
  "?token=" +
  token +
  "&expires=" +
  expires;

console.log(url);
Advanced: token.js;

var queryString = require("querystring");
var crypto = require("crypto");

function addCountries(url, a, b) {
  var tempUrl = url;
  if (a != null) {
    var tempUrlOne = new URL(tempUrl);
    tempUrl += (tempUrlOne.search == "" ? "?" : "&") + "token_countries=" + a;
  }
  if (b != null) {
    var tempUrlTwo = new URL(tempUrl);
    tempUrl +=
      (tempUrlTwo.search == "" ? "?" : "&") + "token_countries_blocked=" + b;
  }
  return tempUrl;
}

function signUrl(
  url,
  securityKey,
  expirationTime = 3600,
  userIp,
  isDirectory = false,
  pathAllowed,
  countriesAllowed,
  countriesBlocked
) {
  /*
        url: CDN URL w/o the trailing '/' - exp. http://test.b-cdn.net/file.png
        securityKey: Security token found in your pull zone
        expirationTime: Authentication validity (default. 86400 sec/24 hrs)
        userIp: Optional parameter if you have the User IP feature enabled
        isDirectory: Optional parameter - "true" returns a URL separated by forward slashes (exp. (domain)/bcdn_token=...)
        pathAllowed: Directory to authenticate (exp. /path/to/images)
        countriesAllowed: List of countries allowed (exp. CA, US, TH)
        countriesBlocked: List of countries blocked (exp. CA, US, TH)
    */
  var parameterData = "",
    parameterDataUrl = "",
    signaturePath = "",
    hashableBase = "",
    token = "";
  var expires = Math.floor(new Date() / 1000) + expirationTime;
  var url = addCountries(url, countriesAllowed, countriesBlocked);
  var parsedUrl = new URL(url);
  var parameters = new URL(url).searchParams;
  if (pathAllowed != "") {
    signaturePath = pathAllowed;
    parameters.set("token_path", signaturePath);
  } else {
    signaturePath = decodeURIComponent(parsedUrl.pathname);
  }
  parameters.sort();
  if (Array.from(parameters).length > 0) {
    parameters.forEach(function (value, key) {
      if (value == "") {
        return;
      }
      if (parameterData.length > 0) {
        parameterData += "&";
      }
      parameterData += key + "=" + value;
      parameterDataUrl += "&" + key + "=" + queryString.escape(value);
    });
  }
  hashableBase =
    securityKey +
    signaturePath +
    expires +
    (userIp != null ? userIp : "") +
    parameterData;
  token = Buffer.from(
    crypto.createHash("sha256").update(hashableBase).digest()
  ).toString("base64");
  token = token
    .replace(/\n/g, "")
    .replace(/\+/g, "-")
    .replace(/\//g, "_")
    .replace(/=/g, "");
  if (isDirectory) {
    return (
      parsedUrl.protocol +
      "//" +
      parsedUrl.host +
      "/bcdn_token=" +
      token +
      parameterDataUrl +
      "&expires=" +
      expires +
      parsedUrl.pathname
    );
  } else {
    return (
      parsedUrl.protocol +
      "//" +
      parsedUrl.host +
      parsedUrl.pathname +
      "?token=" +
      token +
      parameterDataUrl +
      "&expires=" +
      expires
    );
  }
}

module.exports = { signUrl };
index.js;

// Import the signUrl function from the "token.js" file
const { signUrl } = require("./token");

// securityKey can be found on the "Security" tab of the pull zone in the dashboard
const securityKey = "229248f0-f007-4bf9-ba1f-bbf1b4ad9d40";

// The URL would be the full URL you are signing.
const url = "https://token-tester.b-cdn.net/300kb.jpg";

// The expiration time is set to one hour by default.
const expires = Math.floor(Date.now() / 1000) + 3600;

// The IP of the user, leave blank if not using.
const ip = "";

// Enable or disable the use of path tokens.
const pathTokenEnabled = false;

// If pathTokenEnabled is enabled, supply the path to the current directory.
const pathAllowedRoute = "/";

// List of countries allowed seperated by a comma e.g. gb,se,jp
const countriesAllowed = "";

// List of countries blocked seperated by a comma e.g. gb,se,jp
const countriesBlocked = "";

// Generate signed URL.
console.log(
  signUrl(
    url,
    securityKey,
    expires,
    ip,
    pathTokenEnabled,
    pathAllowedRoute,
    countriesAllowed,
    countriesBlocked
  )
);
```
