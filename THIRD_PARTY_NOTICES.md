# Third-party components

The Apache-2.0 license at the repository root covers Acahti-owned code. It does not
replace the licenses of vendored libraries, JavaScript dependencies, container
images or separately operated services.

- Forgejo: https://forgejo.org/ — retain the license and source obligations of the
  pinned version when distributing it or a modified build.
- Woodpecker CI: https://woodpecker-ci.org/ — see its upstream license and notices.
- PostgreSQL: https://www.postgresql.org/about/licence/.
- Vendored Go dependencies retain their LICENSE files under vendor/.
- Frontend dependencies and versions are recorded in web/package-lock.json.

See versions.env for service versions. Acahti runs these services separately;
contributors must preserve their notices when changing packaging.
