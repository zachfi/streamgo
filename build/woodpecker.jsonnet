// build/woodpecker.jsonnet - Woodpecker CI pipeline (jsonnet → .woodpecker.yml)
//
// Generate .woodpecker.yml via: make ci-pipeline (or make ci).
// Pipeline: compile on every run; on main only: build+push tools image, then build+push app image.
// All steps use the tools image from the registry.
//
local registry = 'reg.dist.svc.cluster.znet:5000';
local toolsImage = registry + '/zachfi/streamgo-ci-tools:latest';

// Woodpecker expects pull as bool; volumes as list of "name:path" strings.
local step(name, image=toolsImage, commands=[], when=null, volumeStrings=null) = (
  { name: name, image: image, pull: true, commands: commands }
  + (if when != null then { when: when } else {})
  + (if volumeStrings != null then { volumes: volumeStrings } else {})
);

local makeStep(name, makeTargets, when=null, volumeStrings=null) = step(
  name,
  toolsImage,
  std.map(function(t) 'make %s' % t, makeTargets),
  when,
  volumeStrings
);

local mainOnly = [{ event: 'push', branch: 'main' }];
// Run on push or pull_request (satisfies linter: "Set an event filter for all steps")
local pushOrPR = [{ event: 'push' }, { event: 'pull_request' }];
// Woodpecker step volumes: list of "name:path" or "host_path:container_path" strings
local dockersockVolume = ['dockersock:/var/run'];

// Compile only - runs on every build (PR and main); explicit event filter for linter
local compile() = makeStep('compile-only', ['compile-only'], pushOrPR);

// Build and push tools image (main only). Needs Docker socket.
local buildTools() = makeStep('build-tools', ['tools-image-build registry=%s' % registry, 'tools-image-push registry=%s' % registry], mainOnly, dockersockVolume);

// Build and push app image (main only). Needs Docker socket.
local buildApp() = makeStep('build-app', ['docker registry=%s' % registry, 'docker-push registry=%s' % registry], mainOnly, dockersockVolume);

// Pipeline: main-only steps first (so tools image is available), then compile, then build-app
local steps = [
  buildTools(),
  compile(),
  buildApp(),
];

local services = [
  {
    name: 'docker',
    image: 'docker:24-dind',
    privileged: true,
    volumes: ['dockersock:/var/run'],
  },
];

// No top-level volumes: Woodpecker only documents step/service-level volumes (string list "name:path").
// Drone-style "volumes: [{ name, temp: {} }]" is invalid and can cause "cannot unmarshal '' to type string".
std.manifestYamlDoc({
  steps: steps,
  services: services,
})
