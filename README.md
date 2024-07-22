![](doc/logo.svg)

An early, scrappy, proof-of-concept toolset to run your own private Endlesss server for multiplayer jamming.

# Purpose

With the closure of *Endlesss Ltd.* in May 2024, the [collaborative music tool](https://web.archive.org/web/20201101012543/http://endlesss.fm/) they had created over the previous 6 years became inoperable, cut off from the cloud services. A community that had sprung up around the *Endlesss* tools and workflow were not satisfied with such a bleak and final outcome for what was, for many of us, a vital piece of creative software. 

**OUROCOSM** ensures that *Endlesss* lives up to its name.

---

This project provides guidance and custom tools to provision a self-contained *Endlesss*-compatible suite of services as well as a client-side tool that makes applying the non-invasive patches required to get *Endlesss Studio* talking to our own servers.

Reasons why you might want to do such a thing:

* **You want to jam with your friends again!** It sucks when companies explode and all their cloud-only tools stop working.
* You want to use Endlesss in a live performance setting where internet may be spotty or absent entirely - ideally we can provision an **OUROCOSM** server on a low-power device such as a Raspberry Pi and have a roving, cloud-free way to do 'local jamming'

---

**OUROCOSM** builds upon the considerable work done on the [**OUROVEON** toolset](https://github.com/Unbundlesss/OUROVEON), a set of deep archival and live broadcast tools written to expand Endlesss' powers.

Naturally, this project is not endorsed or sponsored by Endlesss Ltd. Use responsibly, at your own risk.

<br>

# Current Abilities

At time of writing, the **OUROCOSM** tools are currently in daily active use running a community-owned server.

![](doc/cosmex.png)

There is still work required to offer a data-driven setup experience, containerized deployment examples, further Endlesss features to support. `ocServer`, our API service, supports enough to use the core collaborative jamming features in *Endlesss Studio / VST*. Note that at this time, private server support for the Endlesss iOS app is not possible.

The new client-side tool, `ocConnect`, offers a clean, user-friendly way to deploy server configurations to users. It can boot Studio or any other DAW with the appropriate environment setup to bind to the chosen server.

![](doc/ocConnect.png)

<br> 

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


