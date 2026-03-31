import pluginJs from '@eslint/js';
import pluginJsdoc from 'eslint-plugin-jsdoc';
import pluginVue from 'eslint-plugin-vue';
import globals from 'globals';

export default [
  {ignores: ['**/dist/**']},
  {files: ['vite.config.js'], languageOptions: {globals: globals.node}},
  {files: ['src/**/*.vue'], languageOptions: {globals: {'GIT_VERSION': 'readonly'}}},
  {files: ['**/*.cjs'], languageOptions: {globals: globals.commonjs, sourceType: 'script'}},
  pluginJs.configs.recommended,
  ...pluginVue.configs['flat/recommended'],
  pluginJsdoc.configs['flat/recommended-typescript-flavor'],
  {
    name: 'local overrides',
    rules: {
      'semi': ['error', 'always'],
      // Arrange your attributes in any order you like
      'vue/attributes-order': 0,
      'vue/first-attribute-linebreak': ['off'],
      'vue/html-closing-bracket-newline': ['error', {'multiline': 'never'}],
      'vue/html-closing-bracket-spacing': ['error', {'selfClosingTag': 'never'}],
      'vue/html-indent': ['error', 2, {'attribute': 2}],
      'vue/max-attributes-per-line': [2, {'singleline': 10, 'multiline': 10}],
      // let us call things single words
      'vue/multi-word-component-names': 0,
      'vue/singleline-html-element-content-newline': ['off'],
      'vue/valid-v-slot': ['error', {'allowModifiers': true}],

      // because we haven't implemented modules correctly across most of our code
      'jsdoc/no-undefined-types': 0,
      // because it's the types that are more important, it's still preferred but we don't want
      // all the explicit warnings
      'jsdoc/require-param-description': 0,
      'jsdoc/require-property-description': 0,
      'jsdoc/require-returns-description': 0,
      // tags should have no lines between them (the default), except @example which we allow any
      // number of lines between to allow better separation of these tags
      'jsdoc/tag-lines': ['warn', 'never', {
        'startLines': 1, // this was added in a newer version of the plugin
        'tags': {'example': {lines: 'any'}}
      }]
    },
    settings: {
      jsdoc: {
        // Enable import('foo').Foo syntax
        'mode': 'typescript',
        'tagNamePreference': {
          'returns': 'return'
        },
        'preferredTypes': {
          'object': 'Object',
          'Function': 'function'
        }
      }
    }
  },
];
