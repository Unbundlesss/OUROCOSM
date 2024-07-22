# ocConnect

Client launch tool for connecting to private Endlesss servers. Offers a server browser, reachability testing and a simple way to specify and launch plugin hosts to use the Endlesss VST with private servers.

Currently, building the tool requires you to bundle in a suitable `ourocosm.client.yaml` in `/assets/` listing the private servers to give the users access to. 

# Mechanism

Endlesss Studio examines a series of undocumented environment variables on startup, originally designed for internal use when testing the application against QA servers.

We can configure these environment variables to convince Studio to talk to our own services instead. `ocConnect` does this for any chosen application, populating the environment with the chosen server's data.

The following environment variables are used:

 * `ENDLESSS_ENV`
   - set to `local`, this switches Studio over to use a custom server environment
 * `ENDLESSS_DATA_URL`
   - the `host:port` of the CouchDB instance to communicate with
 * `ENDLESSS_API_URL`
   - the `host:port` of the Endlesss API server to use
 * `ENDLESSS_WEB_URL`
   - the `host:port` assumed to be the Endlesss website address
 * `ENDLESSS_HTTPS`
   - `true` or `false`, enabling or disabling secure transport connections to the provided servers

