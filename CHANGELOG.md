# Changelog

## [2.2.2](https://github.com/grafana/smtprelay/compare/v2.2.1...v2.2.2) (2025-05-22)


### Dependencies

* **actions:** Bump actions/setup-go from 5.4.0 to 5.5.0 ([#252](https://github.com/grafana/smtprelay/issues/252)) ([cd1f9d2](https://github.com/grafana/smtprelay/commit/cd1f9d2f5cfa02390eb83ad6c1ce4dd05c9a5b95))
* **actions:** bump github/codeql-action from 3.28.17 to 3.28.18 ([#256](https://github.com/grafana/smtprelay/issues/256)) ([b357472](https://github.com/grafana/smtprelay/commit/b3574724fc99a2655ff9e2ad1834dd18dcf2f158))
* **go:** bump github.com/prometheus/common from 0.63.0 to 0.64.0 ([#255](https://github.com/grafana/smtprelay/issues/255)) ([9fcab7c](https://github.com/grafana/smtprelay/commit/9fcab7c6178632a714317b0766406fb97db8d5bf))
* **go:** bump the go-opentelemetry-io group with 4 updates ([#258](https://github.com/grafana/smtprelay/issues/258)) ([e905c7d](https://github.com/grafana/smtprelay/commit/e905c7d47410750019a5a144bd48739d47a69d1a))

## [2.2.1](https://github.com/grafana/smtprelay/compare/v2.2.0...v2.2.1) (2025-05-22)


### Documentation

* **readme:** fix wrong link ([#239](https://github.com/grafana/smtprelay/issues/239)) ([73ce2bd](https://github.com/grafana/smtprelay/commit/73ce2bdc0064128f42d223e0052d60502b74e2db))


### Dependencies

* **actions:** Bump actions/create-github-app-token from 2.0.2 to 2.0.6 ([#246](https://github.com/grafana/smtprelay/issues/246)) ([bf7b0dc](https://github.com/grafana/smtprelay/commit/bf7b0dc2d24ba3d21269af96a3ff572df3a11c69))
* **actions:** Bump github/codeql-action from 3.28.16 to 3.28.17 ([#244](https://github.com/grafana/smtprelay/issues/244)) ([e5126f7](https://github.com/grafana/smtprelay/commit/e5126f7240c4e189ca6e91e167223cb9fe405c1b))
* **actions:** Bump golangci/golangci-lint-action from 7.0.0 to 8.0.0 ([#247](https://github.com/grafana/smtprelay/issues/247)) ([e4dfb2f](https://github.com/grafana/smtprelay/commit/e4dfb2f730f1be4d1dcf3ed4e8bb0aab9b3d31f5))
* **actions:** Update grafana/shared-workflows requirement to 49eed0955ec059569c3eca1b4221fe7741c2b260 ([#254](https://github.com/grafana/smtprelay/issues/254)) ([e85a831](https://github.com/grafana/smtprelay/commit/e85a831347815af0721ac520e72e203d0d9a5a9c))
* **docker:** Bump golang from `7772cb5` to `ef18ee7` ([#250](https://github.com/grafana/smtprelay/issues/250)) ([132f2ea](https://github.com/grafana/smtprelay/commit/132f2ea847a4bad7cec6325db9a8489ed0396cf2))
* **go:** Bump golang.org/x/crypto from 0.37.0 to 0.38.0 ([#249](https://github.com/grafana/smtprelay/issues/249)) ([9bfcccd](https://github.com/grafana/smtprelay/commit/9bfcccdb0021845dd98eab93aed21fb3176b0add))

## [2.2.0](https://github.com/grafana/smtprelay/compare/v2.1.5...v2.2.0) (2025-04-16)


### Features

* Do not hardcode TLS MinVersion and delegate it to Go version ([#230](https://github.com/grafana/smtprelay/issues/230)) ([b28bd65](https://github.com/grafana/smtprelay/commit/b28bd654b4c082c6d842d83df85744e6b35b03ec))


### Dependencies

* **actions:** bump actions/create-github-app-token from 1 to 2 ([#231](https://github.com/grafana/smtprelay/issues/231)) ([c85bf83](https://github.com/grafana/smtprelay/commit/c85bf83c673f06686275e53b6a8b7671d82ade23))
* **actions:** bump aquasecurity/setup-trivy from 0.2.2 to 0.2.3 ([#233](https://github.com/grafana/smtprelay/issues/233)) ([ace295d](https://github.com/grafana/smtprelay/commit/ace295d6935575bcb9917dfe49e08ef76a752772))
* **actions:** bump golangci/golangci-lint-action from 6 to 7 ([#229](https://github.com/grafana/smtprelay/issues/229)) ([6f8494b](https://github.com/grafana/smtprelay/commit/6f8494bb2046aac534b794981bb1b0e4261aa58b))
* **actions:** Update Trivy to 0.58.1 ([#215](https://github.com/grafana/smtprelay/issues/215)) ([bca429f](https://github.com/grafana/smtprelay/commit/bca429fff1643c8d368ba597d8fe7959b51df32b))
* **docker:** bump alpine from 3.20 to 3.21 ([#205](https://github.com/grafana/smtprelay/issues/205)) ([6280ea5](https://github.com/grafana/smtprelay/commit/6280ea5472b72ff4e13528ab4266bd2225079955))
* **docker:** bump golang from 1.23-alpine to 1.24-alpine ([#220](https://github.com/grafana/smtprelay/issues/220)) ([f221f86](https://github.com/grafana/smtprelay/commit/f221f863ecb294c7e58c72506232892efa4eb0ce))
* **go:** bump github.com/prometheus/client_golang from 1.20.5 to 1.21.1 ([#224](https://github.com/grafana/smtprelay/issues/224)) ([3823d7e](https://github.com/grafana/smtprelay/commit/3823d7ee840bc5a7a1a516d827ac0f9eddbee8fb))
* **go:** bump github.com/prometheus/client_golang from 1.21.1 to 1.22.0 ([#234](https://github.com/grafana/smtprelay/issues/234)) ([790cd38](https://github.com/grafana/smtprelay/commit/790cd386cd168b7a64cc8ebb9819cf2c364d8e3f))
* **go:** bump github.com/prometheus/common from 0.60.1 to 0.61.0 ([#204](https://github.com/grafana/smtprelay/issues/204)) ([319033f](https://github.com/grafana/smtprelay/commit/319033fb2b4bf263b94bdd8a8cf928da76ecbdcf))
* **go:** bump github.com/prometheus/common from 0.61.0 to 0.62.0 ([#216](https://github.com/grafana/smtprelay/issues/216)) ([a1dbb12](https://github.com/grafana/smtprelay/commit/a1dbb128911b929b614892443871c29631becacc))
* **go:** bump github.com/prometheus/common from 0.62.0 to 0.63.0 ([#228](https://github.com/grafana/smtprelay/issues/228)) ([6abbd44](https://github.com/grafana/smtprelay/commit/6abbd440a1fff1fb05335b99a31ac434f838b176))
* **go:** bump github.com/stretchr/testify from 1.9.0 to 1.10.0 ([#201](https://github.com/grafana/smtprelay/issues/201)) ([0bc1c95](https://github.com/grafana/smtprelay/commit/0bc1c95638c01fd38b3dfb27a9462dfb2b845fec))
* **go:** bump go.opentelemetry.io/contrib/samplers/jaegerremote ([#198](https://github.com/grafana/smtprelay/issues/198)) ([76ba49c](https://github.com/grafana/smtprelay/commit/76ba49cb794a7f9aedc2f2e5ad137aad82f9efc1))
* **go:** bump go.opentelemetry.io/otel from 1.32.0 to 1.33.0 ([#208](https://github.com/grafana/smtprelay/issues/208)) ([8a500f9](https://github.com/grafana/smtprelay/commit/8a500f9b2ac51ea9bcee1d3615665872bb5c3552))
* **go:** bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc ([#199](https://github.com/grafana/smtprelay/issues/199)) ([7778f42](https://github.com/grafana/smtprelay/commit/7778f4293a78a3a9b4147eda9c3b0e769acfa2dc))
* **go:** bump go.opentelemetry.io/otel/sdk from 1.31.0 to 1.32.0 ([#197](https://github.com/grafana/smtprelay/issues/197)) ([ada354e](https://github.com/grafana/smtprelay/commit/ada354e1723ae05c5ad3b5b71ee7998b3c722a03))
* **go:** bump golang.org/x/crypto from 0.28.0 to 0.29.0 ([#195](https://github.com/grafana/smtprelay/issues/195)) ([0c5d5f4](https://github.com/grafana/smtprelay/commit/0c5d5f43f8b0da2ad1106c0bd0e79bd897ee1638))
* **go:** bump golang.org/x/crypto from 0.29.0 to 0.31.0 ([#206](https://github.com/grafana/smtprelay/issues/206)) ([3d5e3e6](https://github.com/grafana/smtprelay/commit/3d5e3e608f9807f1706327dd2ea5665b37a58cd5))
* **go:** bump golang.org/x/crypto from 0.31.0 to 0.32.0 ([#212](https://github.com/grafana/smtprelay/issues/212)) ([826cde0](https://github.com/grafana/smtprelay/commit/826cde0dd96a7d4e2f4456b03308698114c87473))
* **go:** bump golang.org/x/crypto from 0.32.0 to 0.35.0 ([#223](https://github.com/grafana/smtprelay/issues/223)) ([4deb0e5](https://github.com/grafana/smtprelay/commit/4deb0e5a6aaeffc33a990989b5096b9005b2d94b))
* **go:** bump golang.org/x/crypto from 0.35.0 to 0.36.0 ([#226](https://github.com/grafana/smtprelay/issues/226)) ([45c2b73](https://github.com/grafana/smtprelay/commit/45c2b7330372f22a2c77ccd52db2c25375ec87f7))
* **go:** bump golang.org/x/crypto from 0.36.0 to 0.37.0 ([#232](https://github.com/grafana/smtprelay/issues/232)) ([e62b9dd](https://github.com/grafana/smtprelay/commit/e62b9dd81e78f748def13fb027e08dbf86262c8f))
* **go:** bump golang.org/x/net from 0.34.0 to 0.36.0 ([#227](https://github.com/grafana/smtprelay/issues/227)) ([d70679a](https://github.com/grafana/smtprelay/commit/d70679a5c0def71ecfe29f31430530f0319a7f77))
* **go:** Bump golang.org/x/net from 0.36.0 to 0.38.0 ([#235](https://github.com/grafana/smtprelay/issues/235)) ([b3e4503](https://github.com/grafana/smtprelay/commit/b3e4503b00483ea4537356c025d8d836e87392c3))
* **go:** bump the go-opentelemetry-io group with 3 updates ([#213](https://github.com/grafana/smtprelay/issues/213)) ([0a2dbbb](https://github.com/grafana/smtprelay/commit/0a2dbbb40cab3d7061907eead5eb0dc01266475a))
* **go:** bump the go-opentelemetry-io group with 5 updates ([#217](https://github.com/grafana/smtprelay/issues/217)) ([1b4b5e4](https://github.com/grafana/smtprelay/commit/1b4b5e4da3ce987a9777fd7c4049aee4621bfa61))
* **go:** bump the go-opentelemetry-io group with 5 updates ([#225](https://github.com/grafana/smtprelay/issues/225)) ([870e465](https://github.com/grafana/smtprelay/commit/870e465d44b8c37a96b76b6074f5f54ee5179131))

## [2.1.5](https://github.com/grafana/smtprelay/compare/v2.1.4...v2.1.5) (2024-10-28)


### Bug Fixes

* **auth:** AuthFetch: comparing authenticating user to user from allowed_user file ([46044aa](https://github.com/grafana/smtprelay/commit/46044aa845f33edb3a5f411d2cc5fce5368cf2bf))


### Dependencies

* **go:** bump github.com/prometheus/common from 0.60.0 to 0.60.1 ([#190](https://github.com/grafana/smtprelay/issues/190)) ([92443cd](https://github.com/grafana/smtprelay/commit/92443cd9337d78bdbe8f2cde9d25b58df180b33e))

## [2.1.4](https://github.com/grafana/smtprelay/compare/v2.1.3...v2.1.4) (2024-10-17)


### Dependencies

* **go:** bump github.com/prometheus/client_golang from 1.20.3 to 1.20.5 ([#189](https://github.com/grafana/smtprelay/issues/189)) ([b0a37e2](https://github.com/grafana/smtprelay/commit/b0a37e2b4fdcc5be556f2c2f11dcc660085fdb95))
* **go:** bump github.com/prometheus/common from 0.59.1 to 0.60.0 ([#182](https://github.com/grafana/smtprelay/issues/182)) ([c75e44f](https://github.com/grafana/smtprelay/commit/c75e44fa578b77f0c9636d680cc1cc3026a0b958))
* **go:** bump go.opentelemetry.io/contrib/samplers/jaegerremote ([#188](https://github.com/grafana/smtprelay/issues/188)) ([cf2a270](https://github.com/grafana/smtprelay/commit/cf2a2703080563e8bfbcba560d9891be92a2151c))
* **go:** bump go.opentelemetry.io/otel from 1.29.0 to 1.30.0 ([#175](https://github.com/grafana/smtprelay/issues/175)) ([92fc713](https://github.com/grafana/smtprelay/commit/92fc713b96e76e3028be4168c33f0740cd63cde4))
* **go:** bump go.opentelemetry.io/otel from 1.30.0 to 1.31.0 ([#185](https://github.com/grafana/smtprelay/issues/185)) ([644f587](https://github.com/grafana/smtprelay/commit/644f587350d1490edb200f41a3c8198b03ca4e6b))
* **go:** bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc ([#177](https://github.com/grafana/smtprelay/issues/177)) ([9828d91](https://github.com/grafana/smtprelay/commit/9828d915c5bb440c39476df27794faf8e4550db1))
* **go:** bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc ([#187](https://github.com/grafana/smtprelay/issues/187)) ([d6b499a](https://github.com/grafana/smtprelay/commit/d6b499af8ed35e466d40ee352f77281b846b75dd))

## [2.1.3](https://github.com/grafana/smtprelay/compare/v2.1.2...v2.1.3) (2024-09-06)


### Dependencies

* **docker:** bump golang from 1.22-alpine to 1.23-alpine ([#156](https://github.com/grafana/smtprelay/issues/156)) ([2dd628e](https://github.com/grafana/smtprelay/commit/2dd628edc0f4bb73fab802cc35bd2d20df656152))
* **go:** bump github.com/grafana/pyroscope-go/godeltaprof ([#158](https://github.com/grafana/smtprelay/issues/158)) ([7ecbd62](https://github.com/grafana/smtprelay/commit/7ecbd62bbf968ae2c2a5ec0d41e067086c0e760b))
* **go:** bump github.com/prometheus/client_golang from 1.19.1 to 1.20.3 ([#170](https://github.com/grafana/smtprelay/issues/170)) ([ce4a641](https://github.com/grafana/smtprelay/commit/ce4a641d9250ec3bfd305166a475c50a5cf23381))
* **go:** bump go.opentelemetry.io/otel from 1.28.0 to 1.29.0 ([#161](https://github.com/grafana/smtprelay/issues/161)) ([aeae268](https://github.com/grafana/smtprelay/commit/aeae26841dd3cb8a657bff17ac66f8a503ac8c3b))
* **go:** Bump supported Go version to 1.23.1 ([#173](https://github.com/grafana/smtprelay/issues/173)) ([f0b6d5f](https://github.com/grafana/smtprelay/commit/f0b6d5f1b6bafe112e48f7fa38d277d46ed00926))

## [2.1.2](https://github.com/grafana/smtprelay/compare/v2.1.1...v2.1.2) (2024-07-23)


### Bug Fixes

* **ci:** Attempt to fix deploy workflow ([#153](https://github.com/grafana/smtprelay/issues/153)) ([90a4e02](https://github.com/grafana/smtprelay/commit/90a4e0298922ab96f59fc4b9cadd0132c2901161))

## [2.1.1](https://github.com/grafana/smtprelay/compare/v2.1.0...v2.1.1) (2024-07-23)


### Bug Fixes

* **mod:** Fix module path to /v2 ([#143](https://github.com/grafana/smtprelay/issues/143)) ([c804e46](https://github.com/grafana/smtprelay/commit/c804e46316ed642463a31489044992e097fd72b3))


### Dependencies

* **actions:** bump golangci/golangci-lint-action from 5 to 6 ([ff49647](https://github.com/grafana/smtprelay/commit/ff49647e83452b42618f32a54c5ecd1c556564b2))
* **docker:** alpine from 3.19 to 3.20 ([#138](https://github.com/grafana/smtprelay/issues/138)) ([301397c](https://github.com/grafana/smtprelay/commit/301397c1c63909f5911c3292e913eace8c2e5959))
* **go:** bump github.com/prometheus/client_golang from 1.19.0 to 1.19.1 ([f737c49](https://github.com/grafana/smtprelay/commit/f737c49645567799616ec892729dc936539ecf58))
* **go:** Bump github.com/prometheus/common from 0.53.0 to 0.54.0 ([#139](https://github.com/grafana/smtprelay/issues/139)) ([d3db2f3](https://github.com/grafana/smtprelay/commit/d3db2f325bee3c052c027b90c03884e448cb7ef4))
* **go:** bump github.com/prometheus/common from 0.54.0 to 0.55.0 ([#144](https://github.com/grafana/smtprelay/issues/144)) ([9600621](https://github.com/grafana/smtprelay/commit/9600621ac6019d92134bd55ecc3bcd10c6027f36))
* **go:** bump go.opentelemetry.io/contrib/samplers/jaegerremote ([#150](https://github.com/grafana/smtprelay/issues/150)) ([a0f2b41](https://github.com/grafana/smtprelay/commit/a0f2b41bfc2f11734095fd052fe2b4895b272803))
* **go:** bump go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc ([#145](https://github.com/grafana/smtprelay/issues/145)) ([7fb6df9](https://github.com/grafana/smtprelay/commit/7fb6df9955dcb3ad5cc79a9f4e6687a98a8254e8))
* **go:** bump golang.org/x/crypto from 0.22.0 to 0.23.0 ([0c43641](https://github.com/grafana/smtprelay/commit/0c43641f7e2e4e26a6e924d24968aff6dfc020b3))
* **go:** Bump golang.org/x/crypto from 0.23.0 to 0.24.0 ([#140](https://github.com/grafana/smtprelay/issues/140)) ([7ff89ce](https://github.com/grafana/smtprelay/commit/7ff89ce1048a2932aa8194203b44fac22695e878))
* **go:** bump golang.org/x/crypto from 0.24.0 to 0.25.0 ([#151](https://github.com/grafana/smtprelay/issues/151)) ([513ef63](https://github.com/grafana/smtprelay/commit/513ef6371c6951ea11dadf71e88843fbc82b510b))
* **go:** bump google.golang.org/grpc from 1.64.0 to 1.64.1 ([#152](https://github.com/grafana/smtprelay/issues/152)) ([f60506f](https://github.com/grafana/smtprelay/commit/f60506f5779ce21bf89316ddbb6661fc3a239a43))
* **go:** go.opentelemetry.io/otel from 1.26.0 to 1.27.0 ([#137](https://github.com/grafana/smtprelay/issues/137)) ([c8a88c6](https://github.com/grafana/smtprelay/commit/c8a88c6ce403a0f68190cbf0dc30274b6e803fe2))
