const path = require('path');

module.exports = {
  entry: './src/index.js',
  output: {
    path: path.resolve(__dirname, 'dist'),
    filename: 'cakework.js',
    globalObject: 'this',
    library: {
		type: 'umd',
		name: 'cakework',
	},
  },
};

