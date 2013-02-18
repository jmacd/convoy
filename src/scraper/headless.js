var page = require('webpage').create();
var url = 'http://localhost:8000/start'

page.settings.userAgent =
    'Mozilla/5.0 (Gentoo; Linux x86_64) AppleWebKit/534.34';

page.open(url, function (status) {
    if (status !== 'success') {
        console.log('Unable to access network');
    } else {
        page.evaluate(function () { });
	console.log('Start page evaluated')
    }
});
