var Vue = require('vue')
var VueRouter = require('vue-router')
Vue.use(VueRouter)

var Main = require('./app/main.vue')
var Login = require('./app/login.vue')
var Chain = require('./app/chain.vue')
var Node = require('./app/node.vue')
var Config = require('./app/config.vue')
var Console = require('./app/console.vue')

import axios from "axios";
import {getCookie, setCookie} from "./utils"

var routes = [
  {path: '/', component: Main, redirect: '/node'},
  {path: '/login', component: Login},
  {path: '/chain', component: Chain},
  {path: '/node', component: Node},
  {path: '/node/config', component: Config},
  {path: '/console', component: Console}
]

var router = new VueRouter({routes, linkActiveClass: "active"})

router.beforeEach((to, from, next) => {
  if (to.path === "/login") {
    if (getCookie("bitmark-webgui") !== "") {
      next({path: "/"})
    } else {
      next()
    }
  } else if (getCookie("bitmark-webgui") === "") {
    next({path: "/login", query: {redirect: to.fullPath}})
  } else {
    next()
  }
})

axios.interceptors.response.use(function (response) {
    // Do something with response data
    return response;
  }, function (error) {
    // Do something with response error
    if (error.response && error.response.status === 401) {
      setCookie("bitmark-webgui", "", 0)
      setCookie("bitmark-webgui-network", "", 0)
      setTimeout(()=>{
        router.push("/login")
      }, 0)
    }
    return Promise.reject(error);
  });

var app = new Vue({
  router,
  el: '#main',
  render: h => h(Main)
})
