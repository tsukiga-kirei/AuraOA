/** 为旧版 OA 内置浏览器补齐数组 at，必须先于 Nuxt 路由初始化执行。 */
export default defineNuxtPlugin({
  name: 'array-at-compat',
  order: -100,
  setup() {
    if (typeof Array.prototype.at === 'function') return
    Object.defineProperty(Array.prototype, 'at', {
      configurable: true,
      writable: true,
      value: function (this: unknown[], index: number) {
        'use strict'
        if (this == null) throw new TypeError('Array.prototype.at called on null or undefined')
        const object = Object(this)
        const lengthNumber = +object.length
        const length = Math.min(Math.max(Math.floor(lengthNumber) || 0, 0), 9007199254740991)
        const number = +index
        const integer = Number.isNaN(number) ? 0 : (number < 0 ? Math.ceil(number) : Math.floor(number))
        const position = integer >= 0 ? integer : length + integer
        return position < 0 || position >= length ? undefined : object[position]
      },
    })
  },
})
