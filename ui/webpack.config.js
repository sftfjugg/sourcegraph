const webpack = require("webpack");
const autoprefixer = require("autoprefixer");
const url = require("url");
const UnusedFilesWebpackPlugin = require("unused-files-webpack-plugin").UnusedFilesWebpackPlugin;
const ProgressBarPlugin = require("progress-bar-webpack-plugin");

const production = (process.env.NODE_ENV === "production");

// 'http' scheme is just used to be able to parse the URL.
const devServerAddr = url.parse(`http://${process.env.WEBPACK_DEV_SERVER_ADDR || "localhost:8080"}`)
const publicURL = url.parse(process.env.PUBLIC_WEBPACK_DEV_SERVER_URL || process.env.WEBPACK_DEV_SERVER_URL || "http://localhost:8080");

// Check dev dependencies.
if (!production) {
	if (process.platform === "darwin") {
		try {
			require("fsevents");
		} catch (error) {
			console.warn("WARNING: fsevents not properly installed. This causes a high CPU load when webpack is idle. Run 'npm run dep' to fix.");
		}
	}
}

const plugins = [
	new webpack.NormalModuleReplacementPlugin(/\/iconv-loader$/, "node-noop"),
	new webpack.DefinePlugin({
		"process.env": {
			NODE_ENV: JSON.stringify(process.env.NODE_ENV || "development"),
		},
		"process.getuid": "function() { return 0; }",
		lazyProxyReject: `function() { }`,
	}),
	new webpack.IgnorePlugin(/testdata\//),
	new webpack.IgnorePlugin(/\.json$/),
	new webpack.IgnorePlugin(/\_test\.js$/),

	// This file isn't actually used, but it contains a dynamic import that Webpack complains about.
	new webpack.IgnorePlugin(/\/monaco\.contribution\.js$/),

	new ProgressBarPlugin(),
];

if (production) {
	plugins.push(
		new webpack.optimize.OccurrenceOrderPlugin(),
		new webpack.optimize.DedupePlugin(),
		new webpack.optimize.UglifyJsPlugin({
			compress: {
				warnings: false,
			},
		})
	);
}

const useHot = !production;
if (useHot) {
	plugins.push(
		new webpack.HotModuleReplacementPlugin()
	);
}

plugins.push(new UnusedFilesWebpackPlugin({
	pattern: "web_modules/**/*.*",
	globOptions: {
		ignore: [
			"**/*.d.ts",
			"**/*_test.tsx",
			"**/testutil/**/*.*",
			"**/testdata/**/*.*",
			"**/*.md",
			"**/*.go",
			"web_modules/sourcegraph/api/index.tsx",
		],
	},
}));

var devtool = "source-map";
if (!production) {
	devtool = process.env.WEBPACK_SOURCEMAPS ? "eval-source-map" : "eval";
}

module.exports = {
	name: "browser",
	target: "web",
	cache: true,
	entry: [
		"./web_modules/sourcegraph/init/browser.tsx",
	],
	resolve: {
		modules: [
			`${__dirname}/web_modules`,
			"node_modules",
			`${__dirname}/node_modules/vscode/src`,
		],
		extensions: ['', '.webpack.js', '.web.js', '.ts', '.tsx', '.js'],
	},
	devtool: devtool,
	output: {
		path: `${__dirname}/assets`,
		filename: production ? "[name].[hash].js" : "[name].js",
		chunkFilename: "c-[chunkhash].js",
		sourceMapFilename: "[file].map",
	},
	plugins: plugins,
	module: {
		loaders: [
			{test: /\.tsx?$/, loader: 'ts'},
			{test: /\.json$/, loader: "json"},
			{test: /\.(woff|eot|ttf)$/, loader: "url?name=fonts/[name].[ext]"},
			{test: /\.(svg|png)$/, loader: "url"},
			{test: /\.css$/, exclude: `${__dirname}/node_modules/vscode`, loader: "style!css?sourceMap&modules&importLoaders=1&localIdentName=[name]__[local]___[hash:base64:5]!postcss"},
			{test: /\.css$/, include: `${__dirname}/node_modules/vscode`, loader: "style!css"}, // TODO(sqs): add ?sourceMap
		],
		noParse: /\.min\.js$/,
	},
	ts: {
		compilerOptions: {
			noEmit: false, // tsconfig.json sets this to true to avoid output when running tsc manually
		},
		transpileOnly: true, // type checking is only done as part of linting or testing
  },
	postcss: [require("postcss-modules-values"), autoprefixer({remove: false})],
	devServer: {
		contentBase: `${__dirname}/assets`,
		host: devServerAddr.hostname,
		public: `${publicURL.hostname}:${publicURL.port}`,
		port: devServerAddr.port,
		headers: {"Access-Control-Allow-Origin": "*"},
		noInfo: true,
		quiet: true,
		hot: useHot,
	},
};

if (useHot) {
	module.exports.entry.unshift("webpack/hot/only-dev-server");
	module.exports.entry.unshift("react-hot-loader/patch");
}
if (!production) {
	module.exports.entry.unshift(`webpack-dev-server/client?${publicURL.format()}`);
}
