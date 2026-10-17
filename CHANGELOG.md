# Changelog

## [1.11.0](https://github.com/soerenschneider/sc/compare/v1.10.0...v1.11.0) (2026-08-30)


### Features

* support for adding taskwarrior tasks ([#248](https://github.com/soerenschneider/sc/issues/248)) ([5957d7b](https://github.com/soerenschneider/sc/commit/5957d7b96e8c9b48f44742cf5f08f3125d47767d))


### Bug Fixes

* **deps:** Update dependency go to v1.27.0 ([#232](https://github.com/soerenschneider/sc/issues/232)) ([5fa0b32](https://github.com/soerenschneider/sc/commit/5fa0b328770e893811827850ff68877af4a510c5))
* **deps:** Update module charm.land/bubbles/v2 to v2.2.1 ([#233](https://github.com/soerenschneider/sc/issues/233)) ([46bd142](https://github.com/soerenschneider/sc/commit/46bd14298e879e5ff07c022df35acb87f86a17f6))
* **deps:** Update module charm.land/bubbletea/v2 to v2.0.9 ([#234](https://github.com/soerenschneider/sc/issues/234)) ([17d8790](https://github.com/soerenschneider/sc/commit/17d8790d480119bd611e61a33fd74fd32ca1ed90))
* **deps:** Update module charm.land/lipgloss/v2 to v2.0.6 ([#235](https://github.com/soerenschneider/sc/issues/235)) ([e63b3c4](https://github.com/soerenschneider/sc/commit/e63b3c47e27271f23bb22229f5f955ce47e010d6))
* **deps:** Update module github.com/coreos/go-oidc/v3 to v3.20.0 ([#228](https://github.com/soerenschneider/sc/issues/228)) ([31e6b3c](https://github.com/soerenschneider/sc/commit/31e6b3c5488369aba6f2a29f4ac46752bfd2c7ac))
* **deps:** Update module github.com/jedib0t/go-pretty/v6 to v6.8.3 ([#236](https://github.com/soerenschneider/sc/issues/236)) ([4de1520](https://github.com/soerenschneider/sc/commit/4de152056f05a2d3174b9b392650a8ed0f7c751d))
* **deps:** Update module github.com/yeqown/go-qrcode/v2 to v2.3.0 ([#247](https://github.com/soerenschneider/sc/issues/247)) ([4cd8ba6](https://github.com/soerenschneider/sc/commit/4cd8ba624b866bc5a1316dd3ca17e4159c14cf15))
* **deps:** Update module github.com/yeqown/go-qrcode/writer/terminal to v1.1.3 ([#246](https://github.com/soerenschneider/sc/issues/246)) ([74dfa42](https://github.com/soerenschneider/sc/commit/74dfa420c7faa91bc0ca3ed00431a59916122e17))
* **deps:** Update module golang.design/x/clipboard to v0.9.0 ([#250](https://github.com/soerenschneider/sc/issues/250)) ([519438e](https://github.com/soerenschneider/sc/commit/519438ea22155f88a9ab18170fcc36dadb2c458c))
* **deps:** Update module golang.org/x/crypto to v0.55.0 ([#238](https://github.com/soerenschneider/sc/issues/238)) ([54fe511](https://github.com/soerenschneider/sc/commit/54fe511ff6018cc00bdb121f7b845681467d62da))
* **deps:** Update module golang.org/x/sys to v0.47.0 ([#239](https://github.com/soerenschneider/sc/issues/239)) ([3bf9eff](https://github.com/soerenschneider/sc/commit/3bf9effa4cbbaac1361ef0f04d3644a245b37464))
* **deps:** Update module golang.org/x/term to v0.45.0 ([#240](https://github.com/soerenschneider/sc/issues/240)) ([f58c744](https://github.com/soerenschneider/sc/commit/f58c74450e7a81c2e4f4e5f1618e1b6cee6ac79e))

## [1.10.0](https://github.com/soerenschneider/sc/compare/v1.9.0...v1.10.0) (2026-06-20)


### Features

* improve log query ux ([#226](https://github.com/soerenschneider/sc/issues/226)) ([24014ff](https://github.com/soerenschneider/sc/commit/24014ff3cbefa00bf53e9cfec2e6e1b5362ede09))


### Bug Fixes

* **deps:** Update module charm.land/lipgloss/v2 to v2.0.4 ([#222](https://github.com/soerenschneider/sc/issues/222)) ([16df06f](https://github.com/soerenschneider/sc/commit/16df06f615c620cc829e3b5169d69ce4d1997acc))
* **deps:** Update module github.com/jedib0t/go-pretty/v6 to v6.8.1 ([#221](https://github.com/soerenschneider/sc/issues/221)) ([6994acd](https://github.com/soerenschneider/sc/commit/6994acd58c2f5268557b99581c9ea10a2a3bbf45))
* **deps:** Update module k8s.io/api to v0.36.2 ([#223](https://github.com/soerenschneider/sc/issues/223)) ([d92b3d4](https://github.com/soerenschneider/sc/commit/d92b3d4286c48572b56dfe3e3d2dd4660c6ead77))
* **deps:** Update module k8s.io/apimachinery to v0.36.2 ([#224](https://github.com/soerenschneider/sc/issues/224)) ([a8da13f](https://github.com/soerenschneider/sc/commit/a8da13ff533e494843dab50a2c64f3bd9462f7c8))
* **deps:** Update module k8s.io/client-go to v0.36.2 ([#225](https://github.com/soerenschneider/sc/issues/225)) ([b28cfa0](https://github.com/soerenschneider/sc/commit/b28cfa08d1a2530112431fe5a23c78c692010577))

## [1.9.0](https://github.com/soerenschneider/sc/compare/v1.8.0...v1.9.0) (2026-06-10)


### Features

* add --force flag to force signing a ssh key regardless of expiry date ([2eb27cb](https://github.com/soerenschneider/sc/commit/2eb27cb73e15067454d54c345b41a02ff73cb33a))
* add flag to add torrent from clipboard without confirmation ([00c2355](https://github.com/soerenschneider/sc/commit/00c23550a2f55db144ea7d51c4c814ed33f0cf23))
* add option to only print value instead of key ([dc38776](https://github.com/soerenschneider/sc/commit/dc3877674bbe9e6a218b8d6aebbe19c070f2ce81))
* list and edit vault kv2 secrets ([b3a6b8c](https://github.com/soerenschneider/sc/commit/b3a6b8c21be20d2f21a7546e32bf485bf24d2eb2))
* print dn instead of magnet link if possible ([02ee3f9](https://github.com/soerenschneider/sc/commit/02ee3f9da8f15586139e5553f125b0774027692e))
* save vault kv2 secrets to local disk ([9ddeea7](https://github.com/soerenschneider/sc/commit/9ddeea7363608024829cc4501d5b9533e18f8ff9))


### Bug Fixes

* **deps:** Update dependency go to v1.26.4 ([#203](https://github.com/soerenschneider/sc/issues/203)) ([0f61184](https://github.com/soerenschneider/sc/commit/0f611842ae09667eafe8c08f73ee117658ce59cd))
* **deps:** Update module charm.land/bubbletea/v2 to v2.0.7 ([#214](https://github.com/soerenschneider/sc/issues/214)) ([b16c91c](https://github.com/soerenschneider/sc/commit/b16c91c0b758d5cba41e779bf8f056feb851b07a))
* **deps:** Update module github.com/jedib0t/go-pretty/v6 to v6.8.0 ([#216](https://github.com/soerenschneider/sc/issues/216)) ([2ca62f2](https://github.com/soerenschneider/sc/commit/2ca62f2df302410d34b83421b33ead5902f914ff))
* **deps:** Update module golang.org/x/crypto to v0.52.0 ([#204](https://github.com/soerenschneider/sc/issues/204)) ([0c979e1](https://github.com/soerenschneider/sc/commit/0c979e19ae8803c13f153cc2207cce94f07ea6a1))
* **deps:** Update module golang.org/x/term to v0.43.0 ([#206](https://github.com/soerenschneider/sc/issues/206)) ([d3090a6](https://github.com/soerenschneider/sc/commit/d3090a6a8cd2774f91add4def41d7d77b7f73cbe))
* **deps:** Update module gopkg.in/ini.v1 to v1.67.2 ([#201](https://github.com/soerenschneider/sc/issues/201)) ([b8ff25f](https://github.com/soerenschneider/sc/commit/b8ff25ff5f43803c220c50b80dc44fc121ee11a6))
* **deps:** Update module k8s.io/api to v0.36.1 ([#209](https://github.com/soerenschneider/sc/issues/209)) ([b54be35](https://github.com/soerenschneider/sc/commit/b54be3503a331a075eadd9b01eaf6458e7a9fbfb))
* **deps:** Update module k8s.io/apimachinery to v0.36.1 ([#210](https://github.com/soerenschneider/sc/issues/210)) ([cd93dbb](https://github.com/soerenschneider/sc/commit/cd93dbb403f792b710f5945692cb13ac4e14d649))
* **deps:** Update module k8s.io/client-go to v0.36.1 ([#211](https://github.com/soerenschneider/sc/issues/211)) ([d802840](https://github.com/soerenschneider/sc/commit/d80284068b03f007fd60cf0c86ceabd0e2de572c))

## [1.8.0](https://github.com/soerenschneider/sc/compare/v1.7.0...v1.8.0) (2026-04-25)


### Features

* kv2 secret sync ([#198](https://github.com/soerenschneider/sc/issues/198)) ([ea18808](https://github.com/soerenschneider/sc/commit/ea18808f7990374576d55d6509534e687179b2ff))


### Bug Fixes

* **deps:** Update dependency go to v1.26.2 ([#182](https://github.com/soerenschneider/sc/issues/182)) ([e5a7ce4](https://github.com/soerenschneider/sc/commit/e5a7ce48ce47b499f3926c7855b4e68516c30897))
* **deps:** Update module github.com/charmbracelet/huh to v2 ([#187](https://github.com/soerenschneider/sc/issues/187)) ([b38d4cc](https://github.com/soerenschneider/sc/commit/b38d4cc1e045820f81142fdd5318fab9b6737c03))
* **deps:** Update module github.com/charmbracelet/lipgloss to v2 ([#179](https://github.com/soerenschneider/sc/issues/179)) ([c58c166](https://github.com/soerenschneider/sc/commit/c58c166ec0fef01556987b17df4aeb6ede5bec5b))
* **deps:** Update module github.com/coreos/go-oidc/v3 to v3.18.0 ([#195](https://github.com/soerenschneider/sc/issues/195)) ([394c7d1](https://github.com/soerenschneider/sc/commit/394c7d19638f9bbc9b9501547a60ef6055a708de))
* **deps:** Update module github.com/hashicorp/vault/api to v1.23.0 ([#190](https://github.com/soerenschneider/sc/issues/190)) ([4fd565e](https://github.com/soerenschneider/sc/commit/4fd565ed49b107c1835a7fa1b87234911ce208a2))
* **deps:** Update module github.com/jedib0t/go-pretty/v6 to v6.7.10 ([3f86219](https://github.com/soerenschneider/sc/commit/3f8621937cc756c6926c2dbb5abe707c2815df46))
* **deps:** Update module github.com/jedib0t/go-pretty/v6 to v6.7.10 ([531d564](https://github.com/soerenschneider/sc/commit/531d56476bfb41d1fbb6bccd91338754a21a64e7))
* **deps:** Update module github.com/rs/zerolog to v1.35.1 ([#191](https://github.com/soerenschneider/sc/issues/191)) ([2114654](https://github.com/soerenschneider/sc/commit/21146545457a179f0c209b50c42b923f9623c5f8))
* **deps:** Update module github.com/zalando/go-keyring to v0.2.8 ([#189](https://github.com/soerenschneider/sc/issues/189)) ([064f9d7](https://github.com/soerenschneider/sc/commit/064f9d7fd4ed241e3876e2a6f1442b8199cc2448))
* **deps:** Update module golang.org/x/crypto to v0.50.0 ([#175](https://github.com/soerenschneider/sc/issues/175)) ([bbcb476](https://github.com/soerenschneider/sc/commit/bbcb4760deaba10ac1eb0b3cc1b7837a7b4ae4f4))
* **deps:** Update module golang.org/x/oauth2 to v0.36.0 ([#185](https://github.com/soerenschneider/sc/issues/185)) ([2ea031a](https://github.com/soerenschneider/sc/commit/2ea031a1b9ea079a67d0be7160b3bb080873b0fe))
* **deps:** Update module golang.org/x/sys to v0.43.0 ([#186](https://github.com/soerenschneider/sc/issues/186)) ([98ff804](https://github.com/soerenschneider/sc/commit/98ff8041d8d1d9fcbc161182e4ce981696611737))
* **deps:** Update module golang.org/x/term to v0.42.0 ([#188](https://github.com/soerenschneider/sc/issues/188)) ([0e87ffb](https://github.com/soerenschneider/sc/commit/0e87ffb873fad8c0d91cf682cc8ff2d299f75c39))

## [1.7.0](https://github.com/soerenschneider/sc/compare/v1.6.0...v1.7.0) (2026-02-20)


### Features

* block until credentials are effective ([bf818a4](https://github.com/soerenschneider/sc/commit/bf818a46ef8608ef8a4d9ecc7d31dd42d481fb26))
* clean inactive torrents ([fbfc16d](https://github.com/soerenschneider/sc/commit/fbfc16d62bfe9ab7929d304f24bf44a449a78ffb))
* display and decode qr codes ([bd2c4d0](https://github.com/soerenschneider/sc/commit/bd2c4d0c9b08b4f09a8dbc4cbcc80f8d8eaf087c))
* enable writing to clipboard ([15e1a34](https://github.com/soerenschneider/sc/commit/15e1a34030afcc5c552c53416987767d4a407dc1))
* generate xkcd-style passwords ([c01fa74](https://github.com/soerenschneider/sc/commit/c01fa7480792786a1159897080b28bbea9fcbfe9))
* listen for signal and transport context to cobra ([6113f05](https://github.com/soerenschneider/sc/commit/6113f0506be95c52cff7134eb5a4a8f6dd719e30))
* skip to password field if username already exists ([1d3004f](https://github.com/soerenschneider/sc/commit/1d3004f8cb30dcead8b0c499b583fd2f65e75a2e))
* try to automatically parse magnet link from clipboard ([7d18531](https://github.com/soerenschneider/sc/commit/7d1853117a66056357c8c7c633f7a1635c23c2b3))


### Bug Fixes

* **deps:** Update dependency go to v1.26.0 ([a91bae5](https://github.com/soerenschneider/sc/commit/a91bae53a6fe2cc23c145337007623c22460d2d3))
* **deps:** Update dependency go to v1.26.0 ([5cc25d8](https://github.com/soerenschneider/sc/commit/5cc25d83f258ca93ccbc42fe6b7026a622891fbd))
* **deps:** Update github.com/charmbracelet/huh/spinner digest to bffc99a ([59b8830](https://github.com/soerenschneider/sc/commit/59b8830789478762cbff5e1d401e30549791f4b7))
* **deps:** Update github.com/charmbracelet/huh/spinner digest to bffc99a ([da81c75](https://github.com/soerenschneider/sc/commit/da81c75d38c988f55de1c97116d3596658b5ed52))
* **deps:** Update module github.com/charmbracelet/bubbles to v0.21.1 ([3b7df45](https://github.com/soerenschneider/sc/commit/3b7df455fe76257048024df01279e9153ac103ab))
* **deps:** Update module github.com/charmbracelet/bubbles to v0.21.1 ([1bf556e](https://github.com/soerenschneider/sc/commit/1bf556e9a821561cbd4d37d62e4f2eb4cd3bb56d))
* **deps:** Update module github.com/charmbracelet/bubbles to v1 ([15043d6](https://github.com/soerenschneider/sc/commit/15043d6b712a5bb612ec374bbf1a70ee7b6e7896))
* **deps:** Update module github.com/charmbracelet/bubbles to v1 ([e10283c](https://github.com/soerenschneider/sc/commit/e10283cf5e222c467924b03510a76b4a0c296cfa))
* **deps:** Update module github.com/charmbracelet/huh to v0.8.0 ([ded5307](https://github.com/soerenschneider/sc/commit/ded53072d4722ad92fed218a5c5218622a7e7197))
* **deps:** Update module github.com/charmbracelet/huh to v0.8.0 ([289c241](https://github.com/soerenschneider/sc/commit/289c24164e6b1dde07a464fe1e14d81b3dca8f9a))
* **deps:** Update module github.com/coreos/go-oidc/v3 to v3.17.0 ([e0e6da1](https://github.com/soerenschneider/sc/commit/e0e6da15a743152bbfcdd3fba6a32a836a5ddc61))
* **deps:** Update module github.com/coreos/go-oidc/v3 to v3.17.0 ([5c2bfb9](https://github.com/soerenschneider/sc/commit/5c2bfb9079f0914e2d05651ae2ef01d8f5db3d5a))
* **deps:** Update module github.com/hashicorp/vault/api to v1.22.0 ([1a35dd9](https://github.com/soerenschneider/sc/commit/1a35dd9c61f2ae1d7e4418a5c11d2e460b7e9fb5))
* **deps:** Update module github.com/hashicorp/vault/api to v1.22.0 ([3b0f4c7](https://github.com/soerenschneider/sc/commit/3b0f4c7b9465de2c6655b09e8bda430d665c4a60))
* **deps:** Update module github.com/jedib0t/go-pretty/v6 to v6.7.8 ([2ca742f](https://github.com/soerenschneider/sc/commit/2ca742f2b1ea728e6801def8676753d88e442102))
* **deps:** Update module github.com/jedib0t/go-pretty/v6 to v6.7.8 ([897371f](https://github.com/soerenschneider/sc/commit/897371fd63aa947c409731816a8aa529ee66714f))
* **deps:** Update module github.com/soerenschneider/sc-agent to v1.13.1 ([75c5872](https://github.com/soerenschneider/sc/commit/75c5872c52f6bfa685efecb041baf54bc0bae346))
* **deps:** Update module github.com/soerenschneider/sc-agent to v1.13.1 ([94926d1](https://github.com/soerenschneider/sc/commit/94926d101420d1feb052b4f843858a7fd0eb94ed))
* **deps:** Update module github.com/spf13/afero to v1.15.0 ([7268716](https://github.com/soerenschneider/sc/commit/72687168f5521985a5c8314a2216ca574d678e21))
* **deps:** Update module github.com/spf13/afero to v1.15.0 ([cc7e002](https://github.com/soerenschneider/sc/commit/cc7e002b081444b93b102b4f8562c8c3f7456c1d))
* **deps:** Update module github.com/spf13/cobra to v1.10.2 ([dd5d24c](https://github.com/soerenschneider/sc/commit/dd5d24c1eddb8f6adc4a9135d1dbca3b0cdcc619))
* **deps:** Update module github.com/spf13/cobra to v1.10.2 ([872cc69](https://github.com/soerenschneider/sc/commit/872cc69e9eff131176c42a235d5916b0d6069983))
* **deps:** Update module github.com/spf13/viper to v1.21.0 ([f144f8d](https://github.com/soerenschneider/sc/commit/f144f8d65bf643e40afeba2ea62afa09b0dcf54d))
* **deps:** Update module github.com/spf13/viper to v1.21.0 ([15fc3af](https://github.com/soerenschneider/sc/commit/15fc3af404d4636547a4f61c2d7414805b50eaf2))
* **deps:** Update module golang.org/x/crypto to v0.45.0 [SECURITY] ([423998b](https://github.com/soerenschneider/sc/commit/423998bea2b963512f98ff613b728725e32ec321))
* **deps:** Update module golang.org/x/crypto to v0.45.0 [SECURITY] ([55f58e6](https://github.com/soerenschneider/sc/commit/55f58e6724f3ab2e892400e13a31f0df8565d47c))
* **deps:** Update module golang.org/x/oauth2 to v0.35.0 ([a697299](https://github.com/soerenschneider/sc/commit/a697299df3e9db693acd6b1a27f2b21d6a90b932))
* **deps:** Update module golang.org/x/oauth2 to v0.35.0 ([d7f140a](https://github.com/soerenschneider/sc/commit/d7f140ac9eee00b9b37abffc0d5e38c1dfb05419))
* **deps:** Update module golang.org/x/sys to v0.41.0 ([9431d71](https://github.com/soerenschneider/sc/commit/9431d711b5d6e96489b90b547ae48decdd13c94c))
* **deps:** Update module golang.org/x/sys to v0.41.0 ([2fba4c6](https://github.com/soerenschneider/sc/commit/2fba4c60ee239188b45c777195e46a3856362b04))
* **deps:** Update module golang.org/x/term to v0.40.0 ([6765300](https://github.com/soerenschneider/sc/commit/67653000454aa372f2754001cef125bd3e4f2c99))
* **deps:** Update module golang.org/x/term to v0.40.0 ([4d2fd41](https://github.com/soerenschneider/sc/commit/4d2fd411dad09bdc9423f4cbb89c6fc9460fe02a))
* **deps:** Update module gopkg.in/ini.v1 to v1.67.1 ([004f93d](https://github.com/soerenschneider/sc/commit/004f93d62be8dc3e9b0c4a27b3625dbeb57928eb))
* **deps:** Update module gopkg.in/ini.v1 to v1.67.1 ([37ac8d8](https://github.com/soerenschneider/sc/commit/37ac8d83ef7abd9bcd9307f84dd1c73990f6b10b))
* fix cancelled context in spinner wait time ([c4ef119](https://github.com/soerenschneider/sc/commit/c4ef11937b807a04ee06df010a04868e7b29b2fa))
* fix potential nil pointer access ([a30f431](https://github.com/soerenschneider/sc/commit/a30f431f30e65b49828afc37dc473c471b76b347))

## [1.6.0](https://github.com/soerenschneider/sc/compare/v1.5.0...v1.6.0) (2025-07-08)


### Features

* read profile from env variable ([26e6e1f](https://github.com/soerenschneider/sc/commit/26e6e1f95a721ecfb539050ea5ffdf0bd92c62ba))
* unblock subcommands by running update check concurrently ([92d87e4](https://github.com/soerenschneider/sc/commit/92d87e4b50461b629af83f1925b3a79ea12adfe7))


### Bug Fixes

* fix condition ([0d7ef53](https://github.com/soerenschneider/sc/commit/0d7ef538cf2b95495dc41e909dbd396a494e82d2))

## [1.5.0](https://github.com/soerenschneider/sc/compare/v1.4.0...v1.5.0) (2025-07-02)


### Features

* add command hierarchy for profiles ([c7ec117](https://github.com/soerenschneider/sc/commit/c7ec11745c33ae69edf4d945e77c3297ea728482))
* add support for oidc ([f8c810f](https://github.com/soerenschneider/sc/commit/f8c810f7a15c45304d262d44aa0640e923febf58))
* add support torrents ([bf307d7](https://github.com/soerenschneider/sc/commit/bf307d75f0a6638dfdd111184c1eb9582ea58990))
* add userdata that remembers last used values for commands ([d5274e3](https://github.com/soerenschneider/sc/commit/d5274e3b4ebf6f4f40da0fe023d328ab0ccb020d))
* improve output ([2a78343](https://github.com/soerenschneider/sc/commit/2a78343d3f9d1340a256f34e2e9d1d3422cba816))
* support for linkding api ([1301d64](https://github.com/soerenschneider/sc/commit/1301d643b481de4be2e4532263cdad2b6f531f86))


### Bug Fixes

* actually only delete selected torrents ([e5d4e4f](https://github.com/soerenschneider/sc/commit/e5d4e4f44a706b88be17a137f1033b1cf9750851))
* **deps:** bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 ([b31db04](https://github.com/soerenschneider/sc/commit/b31db04cdf6df337828425314cf68238e98726ee))
* **deps:** bump github.com/go-viper/mapstructure/v2 from 2.2.1 to 2.3.0 ([21ac91b](https://github.com/soerenschneider/sc/commit/21ac91b4d8d2323d02c47745d18ae7a10583807b))
* **deps:** bump github.com/hashicorp/go-retryablehttp ([b549d2e](https://github.com/soerenschneider/sc/commit/b549d2ea56a13c158a47e6f303137514a62b6ce2))
* **deps:** bump github.com/hashicorp/go-retryablehttp from 0.7.7 to 0.7.8 ([abc9165](https://github.com/soerenschneider/sc/commit/abc9165c28fe1af97f70ca66eece78bd9726fc22))
* **deps:** bump golang from 1.24.3 to 1.24.4 ([d39df20](https://github.com/soerenschneider/sc/commit/d39df201a25af783fa5a951ebc50654647decbc5))
* **deps:** bump golang from 1.24.3 to 1.24.4 ([9379c99](https://github.com/soerenschneider/sc/commit/9379c99e98a70ed87f6140ffd04a3cdace6dbb40))
* **deps:** bump golang.org/x/crypto from 0.38.0 to 0.39.0 ([cb1ca8a](https://github.com/soerenschneider/sc/commit/cb1ca8ab82749c903289c8144fbab7c5536e0e26))
* **deps:** bump golang.org/x/crypto from 0.38.0 to 0.39.0 ([d2b3223](https://github.com/soerenschneider/sc/commit/d2b3223644c3180fac51af9ac8924ef1db77b79e))
* fix build on linux(?) ([f5f49c6](https://github.com/soerenschneider/sc/commit/f5f49c698d9c82db8f8ec2ddb0efb19485f07ce2))
* improve ux by marking persistent flag 'addr' required ([b64ba75](https://github.com/soerenschneider/sc/commit/b64ba7590eb63d132ea7c3bf6dd1ecf794391e3a))
* resolve golangci-lint issues ([ab61070](https://github.com/soerenschneider/sc/commit/ab61070add9217b27d5e94ae08ae3c86d3733603))

## [1.4.0](https://github.com/soerenschneider/sc/compare/v1.3.0...v1.4.0) (2025-05-26)


### Features

* add subcommand to query victorialogs ([ba75997](https://github.com/soerenschneider/sc/commit/ba75997a12ca459a7061fb053c254fa361446007))
* add support for configuring healthchecks ([5e292d4](https://github.com/soerenschneider/sc/commit/5e292d40666a0c59aa083b52d67a184b26f796a9))
* add support for victorialogs ([509c83b](https://github.com/soerenschneider/sc/commit/509c83b5f9b663f649e9a1a073fabe575d016750))
* set max table size ([b7d5120](https://github.com/soerenschneider/sc/commit/b7d5120d125c591d45884313ce1de8e8f4a78322))
* support for profiles ([b308eb8](https://github.com/soerenschneider/sc/commit/b308eb811d5878a50804c60ed82fa879c5a62b29))


### Bug Fixes

* bump all dependencies ([ff52cf5](https://github.com/soerenschneider/sc/commit/ff52cf5901e37b8252eaacd852f5f770b22bc57f))
* do not exit prematurely if profile is supplied ([883ef9a](https://github.com/soerenschneider/sc/commit/883ef9afbb0864d6129a610192a8644543722ff4))
* print warning when profile is not found ([afe3df4](https://github.com/soerenschneider/sc/commit/afe3df4d45ea364c0ec5b0b3096cd0a7d289c119))
* update command ([538f2d4](https://github.com/soerenschneider/sc/commit/538f2d42ae52ca7e78a3b45e8bc46a7b1f27f347))

## [1.3.0](https://github.com/soerenschneider/sc/compare/v1.2.2...v1.3.0) (2025-05-18)


### Features

* add basic support for Vault ([9af30b4](https://github.com/soerenschneider/sc/commit/9af30b4815490885f3a4756d564d00ec7588779a))
* add healthcheck ([8aa9c50](https://github.com/soerenschneider/sc/commit/8aa9c5091622c0c30190c8c4a32f266e8402dc20))
* add subcommands to manage vault aws secret engine ([a9c479d](https://github.com/soerenschneider/sc/commit/a9c479de71b951181ba5e0f55e831e735674dc9f))
* add support for vault mfa totp ([1b84bd6](https://github.com/soerenschneider/sc/commit/1b84bd6ac3f3f1f3974c62f21bb2771f375e6daf))


### Bug Fixes

* **deps:** bump github.com/go-jose/go-jose/v4 from 4.0.1 to 4.0.5 ([0eae9c3](https://github.com/soerenschneider/sc/commit/0eae9c3450ed55aac1359fe3836d16f2620d23e2))
* **deps:** bump github.com/go-jose/go-jose/v4 from 4.0.1 to 4.0.5 ([1c21e0c](https://github.com/soerenschneider/sc/commit/1c21e0cac496bdb5fa1f4bbd56b2a4536ae291ff))
* **deps:** bump github.com/jedib0t/go-pretty/v6 from 6.6.5 to 6.6.7 ([39f5fb4](https://github.com/soerenschneider/sc/commit/39f5fb4431dd0a24f2a786a5015a1570226918ea))
* **deps:** bump github.com/jedib0t/go-pretty/v6 from 6.6.5 to 6.6.7 ([cd4f053](https://github.com/soerenschneider/sc/commit/cd4f0538191deee953b18f97d8a945cf9bf844c1))
* **deps:** bump github.com/soerenschneider/sc-agent from 1.8.0 to 1.10.0 ([aa2ec30](https://github.com/soerenschneider/sc/commit/aa2ec300f6ff68ded03114bf6ccad640edb76913))
* **deps:** bump github.com/soerenschneider/sc-agent from 1.8.0 to 1.10.0 ([d1e01d1](https://github.com/soerenschneider/sc/commit/d1e01d103879d0488ed87c2c102af902d7bd6d57))
* **deps:** bump github.com/spf13/afero from 1.11.0 to 1.12.0 ([33ed6dc](https://github.com/soerenschneider/sc/commit/33ed6dc50879ff5de1f67a7fe8b59e0187b432c8))
* **deps:** bump github.com/spf13/afero from 1.11.0 to 1.12.0 ([d424d62](https://github.com/soerenschneider/sc/commit/d424d62566034337bfe391f2bf0486d323c6034e))
* **deps:** bump github.com/spf13/cobra from 1.8.1 to 1.9.1 ([a4761e0](https://github.com/soerenschneider/sc/commit/a4761e08c5bc8fdc69fe3ec60951fac76f5e214a))
* **deps:** bump github.com/spf13/cobra from 1.8.1 to 1.9.1 ([58d7f53](https://github.com/soerenschneider/sc/commit/58d7f53e64ea3dba82a21102ec720e8d0fc0de8b))
* **deps:** bump golang from 1.23.4 to 1.23.6 ([09beb64](https://github.com/soerenschneider/sc/commit/09beb64f3e49bc8bf891f2f52e0655eac86bd54e))
* **deps:** bump golang from 1.23.4 to 1.23.6 ([11a314e](https://github.com/soerenschneider/sc/commit/11a314e40fb9cfda78f2602224c9a193fa0d0459))
* **deps:** bump golang from 1.23.6 to 1.24.3 ([7d56453](https://github.com/soerenschneider/sc/commit/7d56453f607a7130f4d7bfc55e1ce9b6fe7d26f7))
* **deps:** bump golang from 1.23.6 to 1.24.3 ([6e6e972](https://github.com/soerenschneider/sc/commit/6e6e972a666f2e45bc598f0621f97e5f9bd564ae))
* **deps:** bump golang.org/x/net from 0.33.0 to 0.38.0 ([38a4aad](https://github.com/soerenschneider/sc/commit/38a4aad5857ffd56431a7aa35eb5b67a733bb473))
* **deps:** bump golang.org/x/net from 0.33.0 to 0.38.0 ([0ede4ee](https://github.com/soerenschneider/sc/commit/0ede4ee64a9b384e1dffc1fcfe96341d8133facf))
* **deps:** bump golang.org/x/term from 0.28.0 to 0.29.0 ([9ab8f0f](https://github.com/soerenschneider/sc/commit/9ab8f0f95fd22d3641f9dda0cc53e8b3078a0cd7))
* **deps:** bump golang.org/x/term from 0.28.0 to 0.29.0 ([8809f77](https://github.com/soerenschneider/sc/commit/8809f7702e41ff3fc0ef1ae3464c9a5b0fa64695))
* fix api change ([1e06154](https://github.com/soerenschneider/sc/commit/1e061544b77c82f22bc9a54a7a1babbd572908d8))

## [1.2.2](https://github.com/soerenschneider/sc/compare/v1.2.1...v1.2.2) (2025-01-12)


### Bug Fixes

* **deps:** bump github.com/jedib0t/go-pretty/v6 from 6.6.1 to 6.6.5 ([1d742f1](https://github.com/soerenschneider/sc/commit/1d742f17f091718a337c5424afa99cbe58662123))
* **deps:** bump golang from 1.23.2 to 1.23.4 ([65756b0](https://github.com/soerenschneider/sc/commit/65756b04a7be6be2f98130996c81b0116dde98df))
* **deps:** bump golang.org/x/crypto from 0.28.0 to 0.31.0 ([c498c36](https://github.com/soerenschneider/sc/commit/c498c361ed6f1fff872517888c2e917b4c782ec4))
* **deps:** bump golang.org/x/net from 0.29.0 to 0.33.0 ([a470a44](https://github.com/soerenschneider/sc/commit/a470a44e51b6544d4459e1ff9aae95436d48be4c))
* **deps:** bump golang.org/x/term from 0.25.0 to 0.28.0 ([1772e78](https://github.com/soerenschneider/sc/commit/1772e78fcdf0e34ee442f9b76e8187c8231f44dc))

## [1.2.1](https://github.com/soerenschneider/sc/compare/v1.2.0...v1.2.1) (2024-11-08)


### Bug Fixes

* **deps:** bump github.com/soerenschneider/sc-agent from 1.4.0 to 1.8.0 ([2dc18c0](https://github.com/soerenschneider/sc/commit/2dc18c086b2bea1eaa83ff2b658e981647df7fbd))

## [1.2.0](https://github.com/soerenschneider/sc/compare/v1.1.1...v1.2.0) (2024-11-08)


### Features

* **sc-agent:** automatically set default protocol and port for server if not supplied ([832ef50](https://github.com/soerenschneider/sc/commit/832ef5060edea755ff5822112c631a17bff8b2b3))
* **sc-agent:** expand files to support paths using a tilde prefix ([aee1039](https://github.com/soerenschneider/sc/commit/aee10396ae972998f54928f8dbb72d9edb2ca5de))


### Bug Fixes

* **deps:** bump github.com/jedib0t/go-pretty/v6 from 6.6.0 to 6.6.1 ([31e95fb](https://github.com/soerenschneider/sc/commit/31e95fb15c9aeedc48a9099f27afb99692cf6053))

## [1.1.1](https://github.com/soerenschneider/sc/compare/v1.1.0...v1.1.1) (2024-10-11)


### Bug Fixes

* adapt to change ([76ad84c](https://github.com/soerenschneider/sc/commit/76ad84c00c665f4fdb09c38c89cd63ffbd0b8f02))
* **agent:** bump github.com/hashicorp/vault/api from 1.14.0 to 1.15.0 ([8e5d574](https://github.com/soerenschneider/sc/commit/8e5d5746665a00b89b568344e12d1d6e4d2f2046))
* **agent:** bump github.com/jedib0t/go-pretty/v6 from 6.5.9 to 6.6.0 ([fe6c780](https://github.com/soerenschneider/sc/commit/fe6c7803d183eab049f60e30567efdd3eaed6b8d))
* **agent:** bump github.com/soerenschneider/sc-agent from 1.0.1 to 1.4.0 ([346cc25](https://github.com/soerenschneider/sc/commit/346cc252f02fa5b17a2759980c2bb557e1841be6))
* **agent:** bump github.com/soerenschneider/vault-pki-cli ([71e0259](https://github.com/soerenschneider/sc/commit/71e025920c45b56c80963f56b66490c7d012a8b8))
* **agent:** bump github.com/soerenschneider/vault-ssh-cli ([6cf0914](https://github.com/soerenschneider/sc/commit/6cf0914d7e95c4c51a128cdbb7e0f066b2e661f9))
* **agent:** bump golang from 1.23.0 to 1.23.1 ([fd2f173](https://github.com/soerenschneider/sc/commit/fd2f1736c741941208f6db909d0c213e9021048f))
* **agent:** bump golang from 1.23.1 to 1.23.2 ([c49c32e](https://github.com/soerenschneider/sc/commit/c49c32efda644dd25fa47dfe69896cfa6516b7ae))
* **agent:** bump golang.org/x/term from 0.23.0 to 0.24.0 ([5bd6ed8](https://github.com/soerenschneider/sc/commit/5bd6ed838ccbc991f2591f99d11a8e1cdb6e653b))
* **agent:** bump golang.org/x/term from 0.24.0 to 0.25.0 ([c58afca](https://github.com/soerenschneider/sc/commit/c58afca5c38dbd9b571c5b9737f6e78dfb4e7d7b))
* don't require passing a ca cert ([dbb4211](https://github.com/soerenschneider/sc/commit/dbb42116ab03540b670ca794a5a1fafc85b1015e))
* **lint:** check err ([2a92323](https://github.com/soerenschneider/sc/commit/2a9232397d77c3058bb536420c2a0c72b832b712))

## [1.1.0](https://github.com/soerenschneider/sc/compare/v1.0.0...v1.1.0) (2024-09-04)


### Features

* better logging ([909b4b4](https://github.com/soerenschneider/sc/commit/909b4b4235a226953cb388bc3a7300cafc5f518b))


### Bug Fixes

* bump sc-agent to v1.0.1 ([01f1109](https://github.com/soerenschneider/sc/commit/01f11090802f7ac8a2829ee36b7808c196387093))

## 1.0.0 (2024-09-02)


### Features

* add 'version' subcommand ([3554c3b](https://github.com/soerenschneider/sc/commit/3554c3b11ce64e66ff821cd53a2ff5ba6abdd427))
* check for updated releases on start ([f256c2f](https://github.com/soerenschneider/sc/commit/f256c2f44ec675610a8316681ca7db15fef1787c))
* initial support for 'pki' subcommand ([4759e3a](https://github.com/soerenschneider/sc/commit/4759e3a1fa82d1d5cb17fce9e0b0845f287c81ac))


### Bug Fixes

* set min tls version to 1.3 ([11edcfe](https://github.com/soerenschneider/sc/commit/11edcfe616dc69a58474d08481601124aae5f676))
