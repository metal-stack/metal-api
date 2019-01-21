# Usage

This tests are written for the **REST Client** plugin of visual studio code. If you want to use these test, install this plugin and make sure you have a block like this in your user-settings of VSCode.

~~~json
    "rest-client.environmentVariables": {
        "$shared": {},
        "local-metal" : {
            "scheme":"http",
            "host":"localhost:8080"
        },
    }
~~~

You can then select this environment and simply execute the tests.