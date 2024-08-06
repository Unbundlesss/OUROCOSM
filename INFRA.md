# Minimal Infrastructure

A workable Endlesss deployment consists of 3 core services:

 * [CouchDB 3.3.3](https://couchdb.apache.org/)
   - Endlesss is built around the asynchronous, eventually-consistent collaborative database that CouchDB provides, which it communicates with directly.

<br>

 * An S3-compatible storage service of some kind, eg. [MinIO](https://min.io/)
   - Endlesss signs and uploads audio data into S3 using a key/secret pair embedded into the application binary.

<br>

 * Something that can service the minimal set of Endlesss API requests; for **OUROCOSM**, this is implemented by the `ocServer` application, which also offers additional tools for automated setup and data export.
   - Must deal with user login, jam manifests, user profiles.

<br>

