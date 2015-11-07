# Gumshoe - A Private TV Episode Watcher
=====

## What is it
Gumshoe is a utility to watch various channels (IRC only for now) for updates to your
TV watchlist. With a full REST suite of tools, and a simple CLI, to control
operations, it is a powerful tool to manage your TV shows.

Written in Go, you'll need the Go compiler to compile and install this package.

## Easy to setup
1.  Download Gumshoe: <code>go install github.com/ev1lm0nk3y/gumshoe</code>
1.  Start Gumshoe: <code>/usr/local/gumshoe/gumshoe --start</code>
1.  In a browser, navigate [here]("http://localhost:20123")
1.  Configure Gumshoe to your specific setup. More details on the configuration
options can be found in [Configuration Options]("https://github.com/ev1lm0nk3y/gumshoe/www/help/all_configs.md")
1.  Save your configuration, everything should just start.

Go to the status page on your server to ensure everything is running as expected. 
Check the [FAQ](http://github.com/ev1lm0nk3y/gumshoe/www/help/faq.md") if you have issues
or [File a bug](http://github.com/ev1lm0nk3y/gumshoe//issues/new) if the FAQ doesn't help.

## Interface
Once gumshoe is setup and running, you can configure it via REST or the command
line. Here are the commands:

### Get Current Configuration
<pre><code>gumshoe-cli show config</code></pre>
<pre><code>http://localhost:20123/settings</code></pre>
