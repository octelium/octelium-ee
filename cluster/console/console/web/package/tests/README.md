From the console package directory, install Chromium and run the browser tests:

```sh
npx playwright install chromium
npm test
```

The tests start their own Vite server and mount `TimeAgo` with 100 timestamps and the Session overview with Cordium metadata under React StrictMode. They check reference links, navigation, partial references, and missing or invalid extensions. The Agent chat is mounted against an in-page fake agent (`fetch` and `EventSource`) to check the conversation list, search, composer, streaming, approvals, renaming and deletion. A running cluster is not required.
