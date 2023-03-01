const path = require('path');
const CopyWebpackPlugin = require('copy-webpack-plugin');

module.exports = {
  plugins: [
    new CopyWebpackPlugin({patterns: [{from: 'assets', to: 'assets'}, {from: 'src/animate.min.css'}] })
  ],
  entry: {
    scroll: './src/scroll.js',
    index: './src/index.js',
    fa: './src/fa.js',
    count: './src/count.js',
  },
  output: {
    filename: '[name].js',
    path: path.resolve(__dirname, 'dist'),
  },
  stats: 'errors-only',
};