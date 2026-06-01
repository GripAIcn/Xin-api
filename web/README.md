# web

本模板帮助你快速开始使用 Vue 3 + Vite 进行开发。

## 推荐的 IDE 配置

[VS Code](https://code.visualstudio.com/) + [Vue (Official)](https://marketplace.visualstudio.com/items?itemName=Vue.volar)（并禁用 Vetur）。

## 推荐的浏览器配置

- 基于 Chromium 的浏览器（Chrome、Edge、Brave 等）：
  - [Vue.js devtools](https://chromewebstore.google.com/detail/vuejs-devtools/nhdogjmejiglipccpnnnanhbledajbpd)
  - [在 Chrome DevTools 中开启自定义对象格式化](http://bit.ly/object-formatters)
- Firefox：
  - [Vue.js devtools](https://addons.mozilla.org/en-US/firefox/addon/vue-js-devtools/)
  - [在 Firefox DevTools 中开启自定义对象格式化](https://fxdx.dev/firefox-devtools-custom-object-formatters/)

## TS 中 `.vue` 导入的类型支持

TypeScript 默认无法处理 `.vue` 导入的类型信息，因此我们使用 `vue-tsc` 替代 `tsc` CLI 进行类型检查。在编辑器中，我们需要 [Volar](https://marketplace.visualstudio.com/items?itemName=Vue.volar) 来使 TypeScript 语言服务识别 `.vue` 类型。

## 自定义配置

参考 [Vite 配置文档](https://vite.dev/config/)。

## 项目设置

```sh
npm install
```

### 编译并热重载开发

```sh
npm run dev
```

### 类型检查、编译并压缩用于生产环境

```sh
npm run build
```

### 使用 [ESLint](https://eslint.org/) 进行代码检查

```sh
npm run lint
```
