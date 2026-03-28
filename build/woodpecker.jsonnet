// build/woodpecker.jsonnet - Woodpecker CI pipeline (jsonnet → .woodpecker.yml)
//
// Generate .woodpecker.yml via: make ci-pipeline (or make ci).
// Pipeline: compile on every run; on main only: build+push tools image, then build+push app image.
// All steps use the tools image from the registry.
//
local registry = 'reg.dist.svc.cluster.znet:5000';
local toolsImage = registry + '/zachfi/streamgo-ci-tools:latest';

// Woodpecker expects pull as bool; volumes as list of "name:path" strings.
local step(name, image=toolsImage, commands=[], when=null, volumeStrings=null, environment=null) = (
  { name: name, image: image, pull: true, commands: commands }
  + (if when != null then { when: when } else {})
  + (if volumeStrings != null then { volumes: volumeStrings } else {})
  + (if environment != null then { environment: environment } else {})
);

local makeStep(name, makeTargets, when=null, volumeStrings=null, environment=null) = step(
  name,
  toolsImage,
  std.map(function(t) 'make %s' % t, makeTargets),
  when,
  volumeStrings,
  environment
);

local mainOnly = [{ event: 'push', branch: 'main' }];
// Run on push, pull_request, or manual (satisfies linter: "Set an event filter for all steps")
local pushOrPR = [{ event: 'push' }, { event: 'pull_request' }, { event: 'manual' }];

// Connect to DinD over network (no dockersock PVC needed; works with RWO-only storage like local-path).
local dockerEnv = { DOCKER_HOST: 'tcp://docker:2375', DOCKER_TLS_VERIFY: '0' };

// Compile only - runs on every build (PR and main); explicit event filter for linter
local compile() = makeStep('compile-only', ['compile-only'], pushOrPR);

// Build and push tools image (main only). Uses DinD via DOCKER_HOST.
local buildTools() = makeStep('build-tools', ['tools-image-build registry=%s' % registry, 'tools-image-push registry=%s' % registry], mainOnly, null, dockerEnv);

// Build and push app image (main only). Uses DinD via DOCKER_HOST.
local buildApp() = makeStep('build-app', ['docker registry=%s' % registry, 'docker-push registry=%s' % registry], mainOnly, null, dockerEnv);

// Debug: 60s sleep at start so you can kubectl exec into the first pod (e.g. cat /etc/resolv.conf).
// Remove this step once debugging is done.
local debugSleep60 = {
  name: 'debug-sleep-60',
  image: 'alpine:3.19',
  pull: true,
  when: pushOrPR,
  commands: [ 'sleep 60' ],
};

// Explicit clone step (used because skip_clone: true so the 60s sleep can run first).
local cloneStep = {
  name: 'clone',
  image: 'woodpeckerci/plugin-git',
  pull: true,
  when: pushOrPR,
};

// Optional: long sleep for deeper debugging (branch "debug" or manual run). Remove when not needed.
local debugSleep = {
  name: 'debug-sleep',
  image: 'alpine:3.19',
  pull: true,
  when: [ { event: 'push', branch: 'debug' }, { event: 'manual' } ],
  commands: [ 'sleep 600' ],
};

// Pipeline: debug 60s first, then clone, then main-only steps, compile, build-app
local steps = [
  debugSleep60,
  cloneStep,
  buildTools(),
  debugSleep,
  compile(),
  buildApp(),
];

// DinD listens on 2375 (no TLS) so steps can use DOCKER_HOST=tcp://docker:2375 without a shared socket PVC.
local services = [
  {
    name: 'docker',
    image: 'docker:24-dind',
    privileged: true,
    environment: { DOCKER_TLS_CERTDIR: '' },
  },
];

// Skip default clone so the first step is debug-sleep-60 (you can exec into that pod). Clone runs as second step.
local skip_clone = true;

std.manifestYamlDoc({
  skip_clone: skip_clone,
  steps: steps,
  services: services,
})
