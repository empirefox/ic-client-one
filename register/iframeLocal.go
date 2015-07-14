package register

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

var page = []byte(`
<html>
<head>
  <meta charset="utf-8">
  <script>window.Messenger=function(){function t(t,n){var e="";if(arguments.length<2?e="target error - target and name are both required":"object"!=typeof t?e="target error - target itself must be window object":"string"!=typeof n&&(e="target error - target name must be string type"),e)throw new Error(e);this.target=t,this.name=n}function n(t,n){this.targets={},this.name=t,this.listenFunc=[],e=n||e,"string"!=typeof e&&(e=e.toString()),this.initListen()}var e="[PROJECT_NAME]",i="postMessage"in window;return i?t.prototype.send=function(t){this.target.postMessage(e+t,"*")}:t.prototype.send=function(t){var n=window.navigator[e+this.name];if("function"!=typeof n)throw new Error("target callback function is not defined");n(e+t,window)},n.prototype.addTarget=function(n,e){var i=new t(n,e);this.targets[e]=i},n.prototype.initListen=function(){var t=this,n=function(n){"object"==typeof n&&n.data&&(n=n.data),n=n.slice(e.length);for(var i=0;i<t.listenFunc.length;i++)t.listenFunc[i](n)};i?"addEventListener"in document?window.addEventListener("message",n,!1):"attachEvent"in document&&window.attachEvent("onmessage",n):window.navigator[e+this.name]=n},n.prototype.listen=function(t){this.listenFunc.push(t)},n.prototype.clear=function(){this.listenFunc=[]},n.prototype.send=function(t){var n,e=this.targets;for(n in e)e.hasOwnProperty(n)&&e[n].send(t)},n}();</script>
  <script>"use strict";!function(){function e(e){r.targets[t].send(JSON.stringify(e))}function n(e){o&&o.send(e)}var t="regParent",r=new window.Messenger("regFrame","RegProject");r.listen(function(e){n(e)}),r.addTarget(window.parent,t);var o=new window.WebSocket("ws://127.0.0.1:12301/register");o.binaryType="arraybuffer",o.onopen=function(){e({type:"open"})},o.onclose=function(){e({type:"close"})},o.onerror=function(n){e({type:"error",content:n})},o.onmessage=function(n){e({type:"msg",content:JSON.parse(n.data)})}}();</script>
</head>
<body></body>
</html>
`)

func RegIframeProxyPage(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", page)
}
