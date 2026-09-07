const { withAppBuildGradle } = require('expo/config-plugins');

module.exports = function withAndroidSigning(config) {
  return withAppBuildGradle(config, (config) => {
    const marker = '// Chimii release signing';
    let source = config.modResults.contents;
    // Metro's own cache key cannot help if Gradle skips Metro entirely.
    // Declare public environment inputs for both up-to-date and build-cache checks.
    const bundleMarker = '// Chimii bundle environment inputs';
    if (!source.includes(bundleMarker)) {
      source += `\n${bundleMarker}
tasks.withType(com.facebook.react.tasks.BundleHermesCTask).configureEach {
    inputs.property('chimiiAppEnv', System.getenv('APP_ENV') ?: 'development')
    inputs.property('chimiiApiUrl', System.getenv('EXPO_PUBLIC_API_URL') ?: '')
    inputs.property('chimiiWebUrl', System.getenv('EXPO_PUBLIC_WEB_URL') ?: '')
}
`;
    }
    config.modResults.contents = source;
    if (source.includes(marker)) return config;
    const target = 'signingConfig signingConfigs.debug';
    const release = source.indexOf('        release {');
    const position = source.indexOf(target, release);
    if (release < 0 || position < 0) throw new Error('Expo release signing template changed');
    source = source.slice(0, position) + 'signingConfig signingConfigs.release' + source.slice(position + target.length);
    source = source.replace('    signingConfigs {', `    signingConfigs {
        ${marker}
        release {
            def keyPath = System.getenv('ANDROID_KEYSTORE_PATH')
            if (keyPath) {
                storeFile file(keyPath)
                storePassword System.getenv('ANDROID_KEYSTORE_PASSWORD')
                keyAlias System.getenv('ANDROID_KEY_ALIAS')
                keyPassword System.getenv('ANDROID_KEY_PASSWORD')
            }
        }`);
    config.modResults.contents = source;
    return config;
  });
};
