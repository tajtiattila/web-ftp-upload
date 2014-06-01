
Uploader
========

Simple uploader using Dropzonejs. Users can uploaded files easily after entering
a name, which is used to avoid conflicts with files from others. The files uploaded
to the server will be transferred to the ftp server specified in the config file.

Tested on:
- Ubuntu 12.04 LTS (standalone and behind nginx 1.6 reverse proxy)
- Windows 7 (standalone)

Warning
-------

This project provides no security measures whatsoever, anyone who can access
the webpage could upload any amount of data, flooding the cache directory on
the website, and the upload directory on the specified ftp server.

Usage
-----

Sample config.json:

	{
		"Title": {
			"en": "My awesome uploader"
		},
		"FTPUrl":"ftp://uploader:secret@ftp.example.com/upload"
	}

The service files for use with daemontools are included.

To use it behind nginx, add the following to your site configuration:

	# proxy uploader
	location /my-secret-uploader/ {

		proxy_pass http://unix:/tmp/web-ftp-uploader.socket;

		# websocket support
		proxy_http_version 1.1;
		proxy_set_header Upgrade $http_upgrade;
		proxy_set_header Connection $connection_upgrade;
		proxy_set_header Host $host;
		proxy_read_timeout 120s;
		proxy_redirect off;
		proxy_buffering off; # Optional
	}

Make sure `$connection_upgrade` is specified, for example with the following
contents inside conf.d/websocket.conf:

	# websocket config for proxying
	map $http_upgrade $connection_upgrade {
			default upgrade;
			''      close;
	}

