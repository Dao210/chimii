const { withAppBuildGradle } = require('expo/config-plugins');

module.exports = function withAndroidSigning(config) {
  return withAppBuildGradle(config, (config) => {
    const marker = '// Chimii release signing';
    let source = config.modResults.contents;
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
