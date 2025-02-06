"use strict";

var _templateObject;
function _taggedTemplateLiteral(strings, raw) { if (!raw) { raw = strings.slice(0); } return Object.freeze(Object.defineProperties(strings, { raw: { value: Object.freeze(raw) } })); }
function _regeneratorRuntime() { "use strict"; /*! regenerator-runtime -- Copyright (c) 2014-present, Facebook, Inc. -- license (MIT): https://github.com/facebook/regenerator/blob/main/LICENSE */ _regeneratorRuntime = function _regeneratorRuntime() { return exports; }; var exports = {}, Op = Object.prototype, hasOwn = Op.hasOwnProperty, defineProperty = Object.defineProperty || function (obj, key, desc) { obj[key] = desc.value; }, $Symbol = "function" == typeof Symbol ? Symbol : {}, iteratorSymbol = $Symbol.iterator || "@@iterator", asyncIteratorSymbol = $Symbol.asyncIterator || "@@asyncIterator", toStringTagSymbol = $Symbol.toStringTag || "@@toStringTag"; function define(obj, key, value) { return Object.defineProperty(obj, key, { value: value, enumerable: !0, configurable: !0, writable: !0 }), obj[key]; } try { define({}, ""); } catch (err) { define = function define(obj, key, value) { return obj[key] = value; }; } function wrap(innerFn, outerFn, self, tryLocsList) { var protoGenerator = outerFn && outerFn.prototype instanceof Generator ? outerFn : Generator, generator = Object.create(protoGenerator.prototype), context = new Context(tryLocsList || []); return defineProperty(generator, "_invoke", { value: makeInvokeMethod(innerFn, self, context) }), generator; } function tryCatch(fn, obj, arg) { try { return { type: "normal", arg: fn.call(obj, arg) }; } catch (err) { return { type: "throw", arg: err }; } } exports.wrap = wrap; var ContinueSentinel = {}; function Generator() {} function GeneratorFunction() {} function GeneratorFunctionPrototype() {} var IteratorPrototype = {}; define(IteratorPrototype, iteratorSymbol, function () { return this; }); var getProto = Object.getPrototypeOf, NativeIteratorPrototype = getProto && getProto(getProto(values([]))); NativeIteratorPrototype && NativeIteratorPrototype !== Op && hasOwn.call(NativeIteratorPrototype, iteratorSymbol) && (IteratorPrototype = NativeIteratorPrototype); var Gp = GeneratorFunctionPrototype.prototype = Generator.prototype = Object.create(IteratorPrototype); function defineIteratorMethods(prototype) { ["next", "throw", "return"].forEach(function (method) { define(prototype, method, function (arg) { return this._invoke(method, arg); }); }); } function AsyncIterator(generator, PromiseImpl) { function invoke(method, arg, resolve, reject) { var record = tryCatch(generator[method], generator, arg); if ("throw" !== record.type) { var result = record.arg, value = result.value; return value && "object" == _typeof(value) && hasOwn.call(value, "__await") ? PromiseImpl.resolve(value.__await).then(function (value) { invoke("next", value, resolve, reject); }, function (err) { invoke("throw", err, resolve, reject); }) : PromiseImpl.resolve(value).then(function (unwrapped) { result.value = unwrapped, resolve(result); }, function (error) { return invoke("throw", error, resolve, reject); }); } reject(record.arg); } var previousPromise; defineProperty(this, "_invoke", { value: function value(method, arg) { function callInvokeWithMethodAndArg() { return new PromiseImpl(function (resolve, reject) { invoke(method, arg, resolve, reject); }); } return previousPromise = previousPromise ? previousPromise.then(callInvokeWithMethodAndArg, callInvokeWithMethodAndArg) : callInvokeWithMethodAndArg(); } }); } function makeInvokeMethod(innerFn, self, context) { var state = "suspendedStart"; return function (method, arg) { if ("executing" === state) throw new Error("Generator is already running"); if ("completed" === state) { if ("throw" === method) throw arg; return doneResult(); } for (context.method = method, context.arg = arg;;) { var delegate = context.delegate; if (delegate) { var delegateResult = maybeInvokeDelegate(delegate, context); if (delegateResult) { if (delegateResult === ContinueSentinel) continue; return delegateResult; } } if ("next" === context.method) context.sent = context._sent = context.arg;else if ("throw" === context.method) { if ("suspendedStart" === state) throw state = "completed", context.arg; context.dispatchException(context.arg); } else "return" === context.method && context.abrupt("return", context.arg); state = "executing"; var record = tryCatch(innerFn, self, context); if ("normal" === record.type) { if (state = context.done ? "completed" : "suspendedYield", record.arg === ContinueSentinel) continue; return { value: record.arg, done: context.done }; } "throw" === record.type && (state = "completed", context.method = "throw", context.arg = record.arg); } }; } function maybeInvokeDelegate(delegate, context) { var methodName = context.method, method = delegate.iterator[methodName]; if (undefined === method) return context.delegate = null, "throw" === methodName && delegate.iterator["return"] && (context.method = "return", context.arg = undefined, maybeInvokeDelegate(delegate, context), "throw" === context.method) || "return" !== methodName && (context.method = "throw", context.arg = new TypeError("The iterator does not provide a '" + methodName + "' method")), ContinueSentinel; var record = tryCatch(method, delegate.iterator, context.arg); if ("throw" === record.type) return context.method = "throw", context.arg = record.arg, context.delegate = null, ContinueSentinel; var info = record.arg; return info ? info.done ? (context[delegate.resultName] = info.value, context.next = delegate.nextLoc, "return" !== context.method && (context.method = "next", context.arg = undefined), context.delegate = null, ContinueSentinel) : info : (context.method = "throw", context.arg = new TypeError("iterator result is not an object"), context.delegate = null, ContinueSentinel); } function pushTryEntry(locs) { var entry = { tryLoc: locs[0] }; 1 in locs && (entry.catchLoc = locs[1]), 2 in locs && (entry.finallyLoc = locs[2], entry.afterLoc = locs[3]), this.tryEntries.push(entry); } function resetTryEntry(entry) { var record = entry.completion || {}; record.type = "normal", delete record.arg, entry.completion = record; } function Context(tryLocsList) { this.tryEntries = [{ tryLoc: "root" }], tryLocsList.forEach(pushTryEntry, this), this.reset(!0); } function values(iterable) { if (iterable) { var iteratorMethod = iterable[iteratorSymbol]; if (iteratorMethod) return iteratorMethod.call(iterable); if ("function" == typeof iterable.next) return iterable; if (!isNaN(iterable.length)) { var i = -1, next = function next() { for (; ++i < iterable.length;) if (hasOwn.call(iterable, i)) return next.value = iterable[i], next.done = !1, next; return next.value = undefined, next.done = !0, next; }; return next.next = next; } } return { next: doneResult }; } function doneResult() { return { value: undefined, done: !0 }; } return GeneratorFunction.prototype = GeneratorFunctionPrototype, defineProperty(Gp, "constructor", { value: GeneratorFunctionPrototype, configurable: !0 }), defineProperty(GeneratorFunctionPrototype, "constructor", { value: GeneratorFunction, configurable: !0 }), GeneratorFunction.displayName = define(GeneratorFunctionPrototype, toStringTagSymbol, "GeneratorFunction"), exports.isGeneratorFunction = function (genFun) { var ctor = "function" == typeof genFun && genFun.constructor; return !!ctor && (ctor === GeneratorFunction || "GeneratorFunction" === (ctor.displayName || ctor.name)); }, exports.mark = function (genFun) { return Object.setPrototypeOf ? Object.setPrototypeOf(genFun, GeneratorFunctionPrototype) : (genFun.__proto__ = GeneratorFunctionPrototype, define(genFun, toStringTagSymbol, "GeneratorFunction")), genFun.prototype = Object.create(Gp), genFun; }, exports.awrap = function (arg) { return { __await: arg }; }, defineIteratorMethods(AsyncIterator.prototype), define(AsyncIterator.prototype, asyncIteratorSymbol, function () { return this; }), exports.AsyncIterator = AsyncIterator, exports.async = function (innerFn, outerFn, self, tryLocsList, PromiseImpl) { void 0 === PromiseImpl && (PromiseImpl = Promise); var iter = new AsyncIterator(wrap(innerFn, outerFn, self, tryLocsList), PromiseImpl); return exports.isGeneratorFunction(outerFn) ? iter : iter.next().then(function (result) { return result.done ? result.value : iter.next(); }); }, defineIteratorMethods(Gp), define(Gp, toStringTagSymbol, "Generator"), define(Gp, iteratorSymbol, function () { return this; }), define(Gp, "toString", function () { return "[object Generator]"; }), exports.keys = function (val) { var object = Object(val), keys = []; for (var key in object) keys.push(key); return keys.reverse(), function next() { for (; keys.length;) { var key = keys.pop(); if (key in object) return next.value = key, next.done = !1, next; } return next.done = !0, next; }; }, exports.values = values, Context.prototype = { constructor: Context, reset: function reset(skipTempReset) { if (this.prev = 0, this.next = 0, this.sent = this._sent = undefined, this.done = !1, this.delegate = null, this.method = "next", this.arg = undefined, this.tryEntries.forEach(resetTryEntry), !skipTempReset) for (var name in this) "t" === name.charAt(0) && hasOwn.call(this, name) && !isNaN(+name.slice(1)) && (this[name] = undefined); }, stop: function stop() { this.done = !0; var rootRecord = this.tryEntries[0].completion; if ("throw" === rootRecord.type) throw rootRecord.arg; return this.rval; }, dispatchException: function dispatchException(exception) { if (this.done) throw exception; var context = this; function handle(loc, caught) { return record.type = "throw", record.arg = exception, context.next = loc, caught && (context.method = "next", context.arg = undefined), !!caught; } for (var i = this.tryEntries.length - 1; i >= 0; --i) { var entry = this.tryEntries[i], record = entry.completion; if ("root" === entry.tryLoc) return handle("end"); if (entry.tryLoc <= this.prev) { var hasCatch = hasOwn.call(entry, "catchLoc"), hasFinally = hasOwn.call(entry, "finallyLoc"); if (hasCatch && hasFinally) { if (this.prev < entry.catchLoc) return handle(entry.catchLoc, !0); if (this.prev < entry.finallyLoc) return handle(entry.finallyLoc); } else if (hasCatch) { if (this.prev < entry.catchLoc) return handle(entry.catchLoc, !0); } else { if (!hasFinally) throw new Error("try statement without catch or finally"); if (this.prev < entry.finallyLoc) return handle(entry.finallyLoc); } } } }, abrupt: function abrupt(type, arg) { for (var i = this.tryEntries.length - 1; i >= 0; --i) { var entry = this.tryEntries[i]; if (entry.tryLoc <= this.prev && hasOwn.call(entry, "finallyLoc") && this.prev < entry.finallyLoc) { var finallyEntry = entry; break; } } finallyEntry && ("break" === type || "continue" === type) && finallyEntry.tryLoc <= arg && arg <= finallyEntry.finallyLoc && (finallyEntry = null); var record = finallyEntry ? finallyEntry.completion : {}; return record.type = type, record.arg = arg, finallyEntry ? (this.method = "next", this.next = finallyEntry.finallyLoc, ContinueSentinel) : this.complete(record); }, complete: function complete(record, afterLoc) { if ("throw" === record.type) throw record.arg; return "break" === record.type || "continue" === record.type ? this.next = record.arg : "return" === record.type ? (this.rval = this.arg = record.arg, this.method = "return", this.next = "end") : "normal" === record.type && afterLoc && (this.next = afterLoc), ContinueSentinel; }, finish: function finish(finallyLoc) { for (var i = this.tryEntries.length - 1; i >= 0; --i) { var entry = this.tryEntries[i]; if (entry.finallyLoc === finallyLoc) return this.complete(entry.completion, entry.afterLoc), resetTryEntry(entry), ContinueSentinel; } }, "catch": function _catch(tryLoc) { for (var i = this.tryEntries.length - 1; i >= 0; --i) { var entry = this.tryEntries[i]; if (entry.tryLoc === tryLoc) { var record = entry.completion; if ("throw" === record.type) { var thrown = record.arg; resetTryEntry(entry); } return thrown; } } throw new Error("illegal catch attempt"); }, delegateYield: function delegateYield(iterable, resultName, nextLoc) { return this.delegate = { iterator: values(iterable), resultName: resultName, nextLoc: nextLoc }, "next" === this.method && (this.arg = undefined), ContinueSentinel; } }, exports; }
function asyncGeneratorStep(gen, resolve, reject, _next, _throw, key, arg) { try { var info = gen[key](arg); var value = info.value; } catch (error) { reject(error); return; } if (info.done) { resolve(value); } else { Promise.resolve(value).then(_next, _throw); } }
function _asyncToGenerator(fn) { return function () { var self = this, args = arguments; return new Promise(function (resolve, reject) { var gen = fn.apply(self, args); function _next(value) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "next", value); } function _throw(err) { asyncGeneratorStep(gen, resolve, reject, _next, _throw, "throw", err); } _next(undefined); }); }; }
function ownKeys(object, enumerableOnly) { var keys = Object.keys(object); if (Object.getOwnPropertySymbols) { var symbols = Object.getOwnPropertySymbols(object); enumerableOnly && (symbols = symbols.filter(function (sym) { return Object.getOwnPropertyDescriptor(object, sym).enumerable; })), keys.push.apply(keys, symbols); } return keys; }
function _objectSpread(target) { for (var i = 1; i < arguments.length; i++) { var source = null != arguments[i] ? arguments[i] : {}; i % 2 ? ownKeys(Object(source), !0).forEach(function (key) { _defineProperty(target, key, source[key]); }) : Object.getOwnPropertyDescriptors ? Object.defineProperties(target, Object.getOwnPropertyDescriptors(source)) : ownKeys(Object(source)).forEach(function (key) { Object.defineProperty(target, key, Object.getOwnPropertyDescriptor(source, key)); }); } return target; }
function _defineProperty(obj, key, value) { key = _toPropertyKey(key); if (key in obj) { Object.defineProperty(obj, key, { value: value, enumerable: true, configurable: true, writable: true }); } else { obj[key] = value; } return obj; }
function _toPropertyKey(arg) { var key = _toPrimitive(arg, "string"); return _typeof(key) === "symbol" ? key : String(key); }
function _toPrimitive(input, hint) { if (_typeof(input) !== "object" || input === null) return input; var prim = input[Symbol.toPrimitive]; if (prim !== undefined) { var res = prim.call(input, hint || "default"); if (_typeof(res) !== "object") return res; throw new TypeError("@@toPrimitive must return a primitive value."); } return (hint === "string" ? String : Number)(input); }
function _typeof(obj) { "@babel/helpers - typeof"; return _typeof = "function" == typeof Symbol && "symbol" == typeof Symbol.iterator ? function (obj) { return typeof obj; } : function (obj) { return obj && "function" == typeof Symbol && obj.constructor === Symbol && obj !== Symbol.prototype ? "symbol" : typeof obj; }, _typeof(obj); }
function _toConsumableArray(arr) { return _arrayWithoutHoles(arr) || _iterableToArray(arr) || _unsupportedIterableToArray(arr) || _nonIterableSpread(); }
function _nonIterableSpread() { throw new TypeError("Invalid attempt to spread non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method."); }
function _iterableToArray(iter) { if (typeof Symbol !== "undefined" && iter[Symbol.iterator] != null || iter["@@iterator"] != null) return Array.from(iter); }
function _arrayWithoutHoles(arr) { if (Array.isArray(arr)) return _arrayLikeToArray(arr); }
function _createForOfIteratorHelper(o, allowArrayLike) { var it = typeof Symbol !== "undefined" && o[Symbol.iterator] || o["@@iterator"]; if (!it) { if (Array.isArray(o) || (it = _unsupportedIterableToArray(o)) || allowArrayLike && o && typeof o.length === "number") { if (it) o = it; var i = 0; var F = function F() {}; return { s: F, n: function n() { if (i >= o.length) return { done: true }; return { done: false, value: o[i++] }; }, e: function e(_e2) { throw _e2; }, f: F }; } throw new TypeError("Invalid attempt to iterate non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method."); } var normalCompletion = true, didErr = false, err; return { s: function s() { it = it.call(o); }, n: function n() { var step = it.next(); normalCompletion = step.done; return step; }, e: function e(_e3) { didErr = true; err = _e3; }, f: function f() { try { if (!normalCompletion && it["return"] != null) it["return"](); } finally { if (didErr) throw err; } } }; }
function _slicedToArray(arr, i) { return _arrayWithHoles(arr) || _iterableToArrayLimit(arr, i) || _unsupportedIterableToArray(arr, i) || _nonIterableRest(); }
function _nonIterableRest() { throw new TypeError("Invalid attempt to destructure non-iterable instance.\nIn order to be iterable, non-array objects must have a [Symbol.iterator]() method."); }
function _unsupportedIterableToArray(o, minLen) { if (!o) return; if (typeof o === "string") return _arrayLikeToArray(o, minLen); var n = Object.prototype.toString.call(o).slice(8, -1); if (n === "Object" && o.constructor) n = o.constructor.name; if (n === "Map" || n === "Set") return Array.from(o); if (n === "Arguments" || /^(?:Ui|I)nt(?:8|16|32)(?:Clamped)?Array$/.test(n)) return _arrayLikeToArray(o, minLen); }
function _arrayLikeToArray(arr, len) { if (len == null || len > arr.length) len = arr.length; for (var i = 0, arr2 = new Array(len); i < len; i++) arr2[i] = arr[i]; return arr2; }
function _iterableToArrayLimit(arr, i) { var _i = null == arr ? null : "undefined" != typeof Symbol && arr[Symbol.iterator] || arr["@@iterator"]; if (null != _i) { var _s, _e, _x, _r, _arr = [], _n = !0, _d = !1; try { if (_x = (_i = _i.call(arr)).next, 0 === i) { if (Object(_i) !== _i) return; _n = !1; } else for (; !(_n = (_s = _x.call(_i)).done) && (_arr.push(_s.value), _arr.length !== i); _n = !0); } catch (err) { _d = !0, _e = err; } finally { try { if (!_n && null != _i["return"] && (_r = _i["return"](), Object(_r) !== _r)) return; } finally { if (_d) throw _e; } } return _arr; } }
function _arrayWithHoles(arr) { if (Array.isArray(arr)) return arr; }
(function () {
  // packages/alpinejs/src/scheduler.js
  var flushPending = false;
  var flushing = false;
  var queue = [];
  var lastFlushedIndex = -1;
  function _scheduler(callback) {
    queueJob(callback);
  }
  function queueJob(job) {
    if (!queue.includes(job)) queue.push(job);
    queueFlush();
  }
  function dequeueJob(job) {
    var index = queue.indexOf(job);
    if (index !== -1 && index > lastFlushedIndex) queue.splice(index, 1);
  }
  function queueFlush() {
    if (!flushing && !flushPending) {
      flushPending = true;
      queueMicrotask(flushJobs);
    }
  }
  function flushJobs() {
    flushPending = false;
    flushing = true;
    for (var i = 0; i < queue.length; i++) {
      queue[i]();
      lastFlushedIndex = i;
    }
    queue.length = 0;
    lastFlushedIndex = -1;
    flushing = false;
  }

  // packages/alpinejs/src/reactivity.js
  var reactive;
  var effect;
  var release;
  var raw;
  var shouldSchedule = true;
  function disableEffectScheduling(callback) {
    shouldSchedule = false;
    callback();
    shouldSchedule = true;
  }
  function setReactivityEngine(engine) {
    reactive = engine.reactive;
    release = engine.release;
    effect = function effect(callback) {
      return engine.effect(callback, {
        scheduler: function scheduler(task) {
          if (shouldSchedule) {
            _scheduler(task);
          } else {
            task();
          }
        }
      });
    };
    raw = engine.raw;
  }
  function overrideEffect(override) {
    effect = override;
  }
  function elementBoundEffect(el) {
    var cleanup2 = function cleanup2() {};
    var wrappedEffect = function wrappedEffect(callback) {
      var effectReference = effect(callback);
      if (!el._x_effects) {
        el._x_effects = /* @__PURE__ */new Set();
        el._x_runEffects = function () {
          el._x_effects.forEach(function (i) {
            return i();
          });
        };
      }
      el._x_effects.add(effectReference);
      cleanup2 = function cleanup2() {
        if (effectReference === void 0) return;
        el._x_effects["delete"](effectReference);
        release(effectReference);
      };
      return effectReference;
    };
    return [wrappedEffect, function () {
      cleanup2();
    }];
  }
  function watch(getter, callback) {
    var firstTime = true;
    var oldValue;
    var effectReference = effect(function () {
      var value = getter();
      JSON.stringify(value);
      if (!firstTime) {
        queueMicrotask(function () {
          callback(value, oldValue);
          oldValue = value;
        });
      } else {
        oldValue = value;
      }
      firstTime = false;
    });
    return function () {
      return release(effectReference);
    };
  }

  // packages/alpinejs/src/mutation.js
  var onAttributeAddeds = [];
  var onElRemoveds = [];
  var onElAddeds = [];
  function onElAdded(callback) {
    onElAddeds.push(callback);
  }
  function onElRemoved(el, callback) {
    if (typeof callback === "function") {
      if (!el._x_cleanups) el._x_cleanups = [];
      el._x_cleanups.push(callback);
    } else {
      callback = el;
      onElRemoveds.push(callback);
    }
  }
  function onAttributesAdded(callback) {
    onAttributeAddeds.push(callback);
  }
  function onAttributeRemoved(el, name, callback) {
    if (!el._x_attributeCleanups) el._x_attributeCleanups = {};
    if (!el._x_attributeCleanups[name]) el._x_attributeCleanups[name] = [];
    el._x_attributeCleanups[name].push(callback);
  }
  function cleanupAttributes(el, names) {
    if (!el._x_attributeCleanups) return;
    Object.entries(el._x_attributeCleanups).forEach(function (_ref) {
      var _ref2 = _slicedToArray(_ref, 2),
        name = _ref2[0],
        value = _ref2[1];
      if (names === void 0 || names.includes(name)) {
        value.forEach(function (i) {
          return i();
        });
        delete el._x_attributeCleanups[name];
      }
    });
  }
  function cleanupElement(el) {
    var _el$_x_effects;
    (_el$_x_effects = el._x_effects) === null || _el$_x_effects === void 0 ? void 0 : _el$_x_effects.forEach(dequeueJob);
    while ((_el$_x_cleanups = el._x_cleanups) !== null && _el$_x_cleanups !== void 0 && _el$_x_cleanups.length) {
      var _el$_x_cleanups;
      el._x_cleanups.pop()();
    }
  }
  var observer = new MutationObserver(onMutate);
  var currentlyObserving = false;
  function startObservingMutations() {
    observer.observe(document, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeOldValue: true
    });
    currentlyObserving = true;
  }
  function stopObservingMutations() {
    flushObserver();
    observer.disconnect();
    currentlyObserving = false;
  }
  var queuedMutations = [];
  function flushObserver() {
    var records = observer.takeRecords();
    queuedMutations.push(function () {
      return records.length > 0 && onMutate(records);
    });
    var queueLengthWhenTriggered = queuedMutations.length;
    queueMicrotask(function () {
      if (queuedMutations.length === queueLengthWhenTriggered) {
        while (queuedMutations.length > 0) queuedMutations.shift()();
      }
    });
  }
  function mutateDom(callback) {
    if (!currentlyObserving) return callback();
    stopObservingMutations();
    var result = callback();
    startObservingMutations();
    return result;
  }
  var isCollecting = false;
  var deferredMutations = [];
  function deferMutations() {
    isCollecting = true;
  }
  function flushAndStopDeferringMutations() {
    isCollecting = false;
    onMutate(deferredMutations);
    deferredMutations = [];
  }
  function onMutate(mutations) {
    if (isCollecting) {
      deferredMutations = deferredMutations.concat(mutations);
      return;
    }
    var addedNodes = /* @__PURE__ */new Set();
    var removedNodes = /* @__PURE__ */new Set();
    var addedAttributes = /* @__PURE__ */new Map();
    var removedAttributes = /* @__PURE__ */new Map();
    var _loop = function _loop() {
      if (mutations[i].target._x_ignoreMutationObserver) return "continue";
      if (mutations[i].type === "childList") {
        mutations[i].addedNodes.forEach(function (node) {
          return node.nodeType === 1 && addedNodes.add(node);
        });
        mutations[i].removedNodes.forEach(function (node) {
          return node.nodeType === 1 && removedNodes.add(node);
        });
      }
      if (mutations[i].type === "attributes") {
        var el = mutations[i].target;
        var name = mutations[i].attributeName;
        var oldValue = mutations[i].oldValue;
        var add2 = function add2() {
          if (!addedAttributes.has(el)) addedAttributes.set(el, []);
          addedAttributes.get(el).push({
            name: name,
            value: el.getAttribute(name)
          });
        };
        var remove = function remove() {
          if (!removedAttributes.has(el)) removedAttributes.set(el, []);
          removedAttributes.get(el).push(name);
        };
        if (el.hasAttribute(name) && oldValue === null) {
          add2();
        } else if (el.hasAttribute(name)) {
          remove();
          add2();
        } else {
          remove();
        }
      }
    };
    for (var i = 0; i < mutations.length; i++) {
      var _ret = _loop();
      if (_ret === "continue") continue;
    }
    removedAttributes.forEach(function (attrs, el) {
      cleanupAttributes(el, attrs);
    });
    addedAttributes.forEach(function (attrs, el) {
      onAttributeAddeds.forEach(function (i) {
        return i(el, attrs);
      });
    });
    var _iterator = _createForOfIteratorHelper(removedNodes),
      _step;
    try {
      var _loop2 = function _loop2() {
        var node = _step.value;
        if (addedNodes.has(node)) return "continue";
        onElRemoveds.forEach(function (i) {
          return i(node);
        });
      };
      for (_iterator.s(); !(_step = _iterator.n()).done;) {
        var _ret2 = _loop2();
        if (_ret2 === "continue") continue;
      }
    } catch (err) {
      _iterator.e(err);
    } finally {
      _iterator.f();
    }
    addedNodes.forEach(function (node) {
      node._x_ignoreSelf = true;
      node._x_ignore = true;
    });
    var _iterator2 = _createForOfIteratorHelper(addedNodes),
      _step2;
    try {
      var _loop3 = function _loop3() {
        var node = _step2.value;
        if (removedNodes.has(node)) return "continue";
        if (!node.isConnected) return "continue";
        delete node._x_ignoreSelf;
        delete node._x_ignore;
        onElAddeds.forEach(function (i) {
          return i(node);
        });
        node._x_ignore = true;
        node._x_ignoreSelf = true;
      };
      for (_iterator2.s(); !(_step2 = _iterator2.n()).done;) {
        var _ret3 = _loop3();
        if (_ret3 === "continue") continue;
      }
    } catch (err) {
      _iterator2.e(err);
    } finally {
      _iterator2.f();
    }
    addedNodes.forEach(function (node) {
      delete node._x_ignoreSelf;
      delete node._x_ignore;
    });
    addedNodes = null;
    removedNodes = null;
    addedAttributes = null;
    removedAttributes = null;
  }

  // packages/alpinejs/src/scope.js
  function scope(node) {
    return mergeProxies(closestDataStack(node));
  }
  function addScopeToNode(node, data2, referenceNode) {
    node._x_dataStack = [data2].concat(_toConsumableArray(closestDataStack(referenceNode || node)));
    return function () {
      node._x_dataStack = node._x_dataStack.filter(function (i) {
        return i !== data2;
      });
    };
  }
  function closestDataStack(node) {
    if (node._x_dataStack) return node._x_dataStack;
    if (typeof ShadowRoot === "function" && node instanceof ShadowRoot) {
      return closestDataStack(node.host);
    }
    if (!node.parentNode) {
      return [];
    }
    return closestDataStack(node.parentNode);
  }
  function mergeProxies(objects) {
    return new Proxy({
      objects: objects
    }, mergeProxyTrap);
  }
  var mergeProxyTrap = {
    ownKeys: function ownKeys(_ref3) {
      var objects = _ref3.objects;
      return Array.from(new Set(objects.flatMap(function (i) {
        return Object.keys(i);
      })));
    },
    has: function has(_ref4, name) {
      var objects = _ref4.objects;
      if (name == Symbol.unscopables) return false;
      return objects.some(function (obj) {
        return Object.prototype.hasOwnProperty.call(obj, name) || Reflect.has(obj, name);
      });
    },
    get: function get(_ref5, name, thisProxy) {
      var objects = _ref5.objects;
      if (name == "toJSON") return collapseProxies;
      return Reflect.get(objects.find(function (obj) {
        return Reflect.has(obj, name);
      }) || {}, name, thisProxy);
    },
    set: function set(_ref6, name, value, thisProxy) {
      var objects = _ref6.objects;
      var target = objects.find(function (obj) {
        return Object.prototype.hasOwnProperty.call(obj, name);
      }) || objects[objects.length - 1];
      var descriptor = Object.getOwnPropertyDescriptor(target, name);
      if (descriptor !== null && descriptor !== void 0 && descriptor.set && descriptor !== null && descriptor !== void 0 && descriptor.get) return descriptor.set.call(thisProxy, value) || true;
      return Reflect.set(target, name, value);
    }
  };
  function collapseProxies() {
    var _this = this;
    var keys = Reflect.ownKeys(this);
    return keys.reduce(function (acc, key) {
      acc[key] = Reflect.get(_this, key);
      return acc;
    }, {});
  }

  // packages/alpinejs/src/interceptor.js
  function initInterceptors(data2) {
    var isObject2 = function isObject2(val) {
      return _typeof(val) === "object" && !Array.isArray(val) && val !== null;
    };
    var recurse = function recurse(obj) {
      var basePath = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : "";
      Object.entries(Object.getOwnPropertyDescriptors(obj)).forEach(function (_ref7) {
        var _ref8 = _slicedToArray(_ref7, 2),
          key = _ref8[0],
          _ref8$ = _ref8[1],
          value = _ref8$.value,
          enumerable = _ref8$.enumerable;
        if (enumerable === false || value === void 0) return;
        if (_typeof(value) === "object" && value !== null && value.__v_skip) return;
        var path = basePath === "" ? key : "".concat(basePath, ".").concat(key);
        if (_typeof(value) === "object" && value !== null && value._x_interceptor) {
          obj[key] = value.initialize(data2, path, key);
        } else {
          if (isObject2(value) && value !== obj && !(value instanceof Element)) {
            recurse(value, path);
          }
        }
      });
    };
    return recurse(data2);
  }
  function interceptor(callback) {
    var mutateObj = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : function () {};
    var obj = {
      initialValue: void 0,
      _x_interceptor: true,
      initialize: function initialize(data2, path, key) {
        return callback(this.initialValue, function () {
          return get(data2, path);
        }, function (value) {
          return set(data2, path, value);
        }, path, key);
      }
    };
    mutateObj(obj);
    return function (initialValue) {
      if (_typeof(initialValue) === "object" && initialValue !== null && initialValue._x_interceptor) {
        var initialize = obj.initialize.bind(obj);
        obj.initialize = function (data2, path, key) {
          var innerValue = initialValue.initialize(data2, path, key);
          obj.initialValue = innerValue;
          return initialize(data2, path, key);
        };
      } else {
        obj.initialValue = initialValue;
      }
      return obj;
    };
  }
  function get(obj, path) {
    return path.split(".").reduce(function (carry, segment) {
      return carry[segment];
    }, obj);
  }
  function set(obj, path, value) {
    if (typeof path === "string") path = path.split(".");
    if (path.length === 1) obj[path[0]] = value;else if (path.length === 0) throw error;else {
      if (obj[path[0]]) return set(obj[path[0]], path.slice(1), value);else {
        obj[path[0]] = {};
        return set(obj[path[0]], path.slice(1), value);
      }
    }
  }

  // packages/alpinejs/src/magics.js
  var magics = {};
  function magic(name, callback) {
    magics[name] = callback;
  }
  function injectMagics(obj, el) {
    var memoizedUtilities = getUtilities(el);
    Object.entries(magics).forEach(function (_ref9) {
      var _ref10 = _slicedToArray(_ref9, 2),
        name = _ref10[0],
        callback = _ref10[1];
      Object.defineProperty(obj, "$".concat(name), {
        get: function get() {
          return callback(el, memoizedUtilities);
        },
        enumerable: false
      });
    });
    return obj;
  }
  function getUtilities(el) {
    var _getElementBoundUtili = getElementBoundUtilities(el),
      _getElementBoundUtili2 = _slicedToArray(_getElementBoundUtili, 2),
      utilities = _getElementBoundUtili2[0],
      cleanup2 = _getElementBoundUtili2[1];
    var utils = _objectSpread({
      interceptor: interceptor
    }, utilities);
    onElRemoved(el, cleanup2);
    return utils;
  }

  // packages/alpinejs/src/utils/error.js
  function tryCatch(el, expression, callback) {
    try {
      for (var _len = arguments.length, args = new Array(_len > 3 ? _len - 3 : 0), _key = 3; _key < _len; _key++) {
        args[_key - 3] = arguments[_key];
      }
      return callback.apply(void 0, args);
    } catch (e) {
      handleError(e, el, expression);
    }
  }
  function handleError(error2, el) {
    var _error;
    var expression = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : void 0;
    error2 = Object.assign((_error = error2) !== null && _error !== void 0 ? _error : {
      message: "No error message given."
    }, {
      el: el,
      expression: expression
    });
    console.warn("Alpine Expression Error: ".concat(error2.message, "\n\n").concat(expression ? 'Expression: "' + expression + '"\n\n' : ""), el);
    setTimeout(function () {
      throw error2;
    }, 0);
  }

  // packages/alpinejs/src/evaluator.js
  var shouldAutoEvaluateFunctions = true;
  function dontAutoEvaluateFunctions(callback) {
    var cache = shouldAutoEvaluateFunctions;
    shouldAutoEvaluateFunctions = false;
    var result = callback();
    shouldAutoEvaluateFunctions = cache;
    return result;
  }
  function evaluate(el, expression) {
    var extras = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : {};
    var result;
    evaluateLater(el, expression)(function (value) {
      return result = value;
    }, extras);
    return result;
  }
  function evaluateLater() {
    return theEvaluatorFunction.apply(void 0, arguments);
  }
  var theEvaluatorFunction = normalEvaluator;
  function setEvaluator(newEvaluator) {
    theEvaluatorFunction = newEvaluator;
  }
  function normalEvaluator(el, expression) {
    var overriddenMagics = {};
    injectMagics(overriddenMagics, el);
    var dataStack = [overriddenMagics].concat(_toConsumableArray(closestDataStack(el)));
    var evaluator = typeof expression === "function" ? generateEvaluatorFromFunction(dataStack, expression) : generateEvaluatorFromString(dataStack, expression, el);
    return tryCatch.bind(null, el, expression, evaluator);
  }
  function generateEvaluatorFromFunction(dataStack, func) {
    return function () {
      var receiver = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : function () {};
      var _ref11 = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : {},
        _ref11$scope = _ref11.scope,
        scope2 = _ref11$scope === void 0 ? {} : _ref11$scope,
        _ref11$params = _ref11.params,
        params = _ref11$params === void 0 ? [] : _ref11$params;
      var result = func.apply(mergeProxies([scope2].concat(_toConsumableArray(dataStack))), params);
      runIfTypeOfFunction(receiver, result);
    };
  }
  var evaluatorMemo = {};
  function generateFunctionFromString(expression, el) {
    if (evaluatorMemo[expression]) {
      return evaluatorMemo[expression];
    }
    var AsyncFunction = Object.getPrototypeOf( /*#__PURE__*/_asyncToGenerator( /*#__PURE__*/_regeneratorRuntime().mark(function _callee() {
      return _regeneratorRuntime().wrap(function _callee$(_context) {
        while (1) switch (_context.prev = _context.next) {
          case 0:
          case "end":
            return _context.stop();
        }
      }, _callee);
    }))).constructor;
    var rightSideSafeExpression = /^[\n\s]*if.*\(.*\)/.test(expression.trim()) || /^(let|const)\s/.test(expression.trim()) ? "(async()=>{ ".concat(expression, " })()") : expression;
    var safeAsyncFunction = function safeAsyncFunction() {
      try {
        var func2 = new AsyncFunction(["__self", "scope"], "with (scope) { __self.result = ".concat(rightSideSafeExpression, " }; __self.finished = true; return __self.result;"));
        Object.defineProperty(func2, "name", {
          value: "[Alpine] ".concat(expression)
        });
        return func2;
      } catch (error2) {
        handleError(error2, el, expression);
        return Promise.resolve();
      }
    };
    var func = safeAsyncFunction();
    evaluatorMemo[expression] = func;
    return func;
  }
  function generateEvaluatorFromString(dataStack, expression, el) {
    var func = generateFunctionFromString(expression, el);
    return function () {
      var receiver = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : function () {};
      var _ref13 = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : {},
        _ref13$scope = _ref13.scope,
        scope2 = _ref13$scope === void 0 ? {} : _ref13$scope,
        _ref13$params = _ref13.params,
        params = _ref13$params === void 0 ? [] : _ref13$params;
      func.result = void 0;
      func.finished = false;
      var completeScope = mergeProxies([scope2].concat(_toConsumableArray(dataStack)));
      if (typeof func === "function") {
        var promise = func(func, completeScope)["catch"](function (error2) {
          return handleError(error2, el, expression);
        });
        if (func.finished) {
          runIfTypeOfFunction(receiver, func.result, completeScope, params, el);
          func.result = void 0;
        } else {
          promise.then(function (result) {
            runIfTypeOfFunction(receiver, result, completeScope, params, el);
          })["catch"](function (error2) {
            return handleError(error2, el, expression);
          })["finally"](function () {
            return func.result = void 0;
          });
        }
      }
    };
  }
  function runIfTypeOfFunction(receiver, value, scope2, params, el) {
    if (shouldAutoEvaluateFunctions && typeof value === "function") {
      var result = value.apply(scope2, params);
      if (result instanceof Promise) {
        result.then(function (i) {
          return runIfTypeOfFunction(receiver, i, scope2, params);
        })["catch"](function (error2) {
          return handleError(error2, el, value);
        });
      } else {
        receiver(result);
      }
    } else if (_typeof(value) === "object" && value instanceof Promise) {
      value.then(function (i) {
        return receiver(i);
      });
    } else {
      receiver(value);
    }
  }

  // packages/alpinejs/src/directives.js
  var prefixAsString = "x-";
  function prefix() {
    var subject = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : "";
    return prefixAsString + subject;
  }
  function setPrefix(newPrefix) {
    prefixAsString = newPrefix;
  }
  var directiveHandlers = {};
  function directive(name, callback) {
    directiveHandlers[name] = callback;
    return {
      before: function before(directive2) {
        if (!directiveHandlers[directive2]) {
          console.warn(String.raw(_templateObject || (_templateObject = _taggedTemplateLiteral(["Cannot find directive `", "`. `", "` will use the default order of execution"], ["Cannot find directive \\`", "\\`. \\`", "\\` will use the default order of execution"])), directive2, name));
          return;
        }
        var pos = directiveOrder.indexOf(directive2);
        directiveOrder.splice(pos >= 0 ? pos : directiveOrder.indexOf("DEFAULT"), 0, name);
      }
    };
  }
  function directiveExists(name) {
    return Object.keys(directiveHandlers).includes(name);
  }
  function directives(el, attributes, originalAttributeOverride) {
    attributes = Array.from(attributes);
    if (el._x_virtualDirectives) {
      var vAttributes = Object.entries(el._x_virtualDirectives).map(function (_ref14) {
        var _ref15 = _slicedToArray(_ref14, 2),
          name = _ref15[0],
          value = _ref15[1];
        return {
          name: name,
          value: value
        };
      });
      var staticAttributes = attributesOnly(vAttributes);
      vAttributes = vAttributes.map(function (attribute) {
        if (staticAttributes.find(function (attr) {
          return attr.name === attribute.name;
        })) {
          return {
            name: "x-bind:".concat(attribute.name),
            value: "\"".concat(attribute.value, "\"")
          };
        }
        return attribute;
      });
      attributes = attributes.concat(vAttributes);
    }
    var transformedAttributeMap = {};
    var directives2 = attributes.map(toTransformedAttributes(function (newName, oldName) {
      return transformedAttributeMap[newName] = oldName;
    })).filter(outNonAlpineAttributes).map(toParsedDirectives(transformedAttributeMap, originalAttributeOverride)).sort(byPriority);
    return directives2.map(function (directive2) {
      return getDirectiveHandler(el, directive2);
    });
  }
  function attributesOnly(attributes) {
    return Array.from(attributes).map(toTransformedAttributes()).filter(function (attr) {
      return !outNonAlpineAttributes(attr);
    });
  }
  var isDeferringHandlers = false;
  var directiveHandlerStacks = /* @__PURE__ */new Map();
  var currentHandlerStackKey = Symbol();
  function deferHandlingDirectives(callback) {
    isDeferringHandlers = true;
    var key = Symbol();
    currentHandlerStackKey = key;
    directiveHandlerStacks.set(key, []);
    var flushHandlers = function flushHandlers() {
      while (directiveHandlerStacks.get(key).length) directiveHandlerStacks.get(key).shift()();
      directiveHandlerStacks["delete"](key);
    };
    var stopDeferring = function stopDeferring() {
      isDeferringHandlers = false;
      flushHandlers();
    };
    callback(flushHandlers);
    stopDeferring();
  }
  function getElementBoundUtilities(el) {
    var cleanups = [];
    var cleanup2 = function cleanup2(callback) {
      return cleanups.push(callback);
    };
    var _elementBoundEffect = elementBoundEffect(el),
      _elementBoundEffect2 = _slicedToArray(_elementBoundEffect, 2),
      effect3 = _elementBoundEffect2[0],
      cleanupEffect = _elementBoundEffect2[1];
    cleanups.push(cleanupEffect);
    var utilities = {
      Alpine: alpine_default,
      effect: effect3,
      cleanup: cleanup2,
      evaluateLater: evaluateLater.bind(evaluateLater, el),
      evaluate: evaluate.bind(evaluate, el)
    };
    var doCleanup = function doCleanup() {
      return cleanups.forEach(function (i) {
        return i();
      });
    };
    return [utilities, doCleanup];
  }
  function getDirectiveHandler(el, directive2) {
    var noop = function noop() {};
    var handler4 = directiveHandlers[directive2.type] || noop;
    var _getElementBoundUtili3 = getElementBoundUtilities(el),
      _getElementBoundUtili4 = _slicedToArray(_getElementBoundUtili3, 2),
      utilities = _getElementBoundUtili4[0],
      cleanup2 = _getElementBoundUtili4[1];
    onAttributeRemoved(el, directive2.original, cleanup2);
    var fullHandler = function fullHandler() {
      if (el._x_ignore || el._x_ignoreSelf) return;
      handler4.inline && handler4.inline(el, directive2, utilities);
      handler4 = handler4.bind(handler4, el, directive2, utilities);
      isDeferringHandlers ? directiveHandlerStacks.get(currentHandlerStackKey).push(handler4) : handler4();
    };
    fullHandler.runCleanups = cleanup2;
    return fullHandler;
  }
  var startingWith = function startingWith(subject, replacement) {
    return function (_ref16) {
      var name = _ref16.name,
        value = _ref16.value;
      if (name.startsWith(subject)) name = name.replace(subject, replacement);
      return {
        name: name,
        value: value
      };
    };
  };
  var into = function into(i) {
    return i;
  };
  function toTransformedAttributes() {
    var callback = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : function () {};
    return function (_ref17) {
      var name = _ref17.name,
        value = _ref17.value;
      var _attributeTransformer = attributeTransformers.reduce(function (carry, transform) {
          return transform(carry);
        }, {
          name: name,
          value: value
        }),
        newName = _attributeTransformer.name,
        newValue = _attributeTransformer.value;
      if (newName !== name) callback(newName, name);
      return {
        name: newName,
        value: newValue
      };
    };
  }
  var attributeTransformers = [];
  function mapAttributes(callback) {
    attributeTransformers.push(callback);
  }
  function outNonAlpineAttributes(_ref18) {
    var name = _ref18.name;
    return alpineAttributeRegex().test(name);
  }
  var alpineAttributeRegex = function alpineAttributeRegex() {
    return new RegExp("^".concat(prefixAsString, "([^:^.]+)\\b"));
  };
  function toParsedDirectives(transformedAttributeMap, originalAttributeOverride) {
    return function (_ref19) {
      var name = _ref19.name,
        value = _ref19.value;
      var typeMatch = name.match(alpineAttributeRegex());
      var valueMatch = name.match(/:([a-zA-Z0-9\-_:]+)/);
      var modifiers = name.match(/\.[^.\]]+(?=[^\]]*$)/g) || [];
      var original = originalAttributeOverride || transformedAttributeMap[name] || name;
      return {
        type: typeMatch ? typeMatch[1] : null,
        value: valueMatch ? valueMatch[1] : null,
        modifiers: modifiers.map(function (i) {
          return i.replace(".", "");
        }),
        expression: value,
        original: original
      };
    };
  }
  var DEFAULT = "DEFAULT";
  var directiveOrder = ["ignore", "ref", "data", "id", "anchor", "bind", "init", "for", "model", "modelable", "transition", "show", "if", DEFAULT, "teleport"];
  function byPriority(a, b) {
    var typeA = directiveOrder.indexOf(a.type) === -1 ? DEFAULT : a.type;
    var typeB = directiveOrder.indexOf(b.type) === -1 ? DEFAULT : b.type;
    return directiveOrder.indexOf(typeA) - directiveOrder.indexOf(typeB);
  }

  // packages/alpinejs/src/utils/dispatch.js
  function dispatch(el, name) {
    var detail = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : {};
    el.dispatchEvent(new CustomEvent(name, {
      detail: detail,
      bubbles: true,
      // Allows events to pass the shadow DOM barrier.
      composed: true,
      cancelable: true
    }));
  }

  // packages/alpinejs/src/utils/walk.js
  function walk(el, callback) {
    if (typeof ShadowRoot === "function" && el instanceof ShadowRoot) {
      Array.from(el.children).forEach(function (el2) {
        return walk(el2, callback);
      });
      return;
    }
    var skip = false;
    callback(el, function () {
      return skip = true;
    });
    if (skip) return;
    var node = el.firstElementChild;
    while (node) {
      walk(node, callback, false);
      node = node.nextElementSibling;
    }
  }

  // packages/alpinejs/src/utils/warn.js
  function warn(message) {
    var _console;
    for (var _len2 = arguments.length, args = new Array(_len2 > 1 ? _len2 - 1 : 0), _key2 = 1; _key2 < _len2; _key2++) {
      args[_key2 - 1] = arguments[_key2];
    }
    (_console = console).warn.apply(_console, ["Alpine Warning: ".concat(message)].concat(args));
  }

  // packages/alpinejs/src/lifecycle.js
  var started = false;
  function start() {
    if (started) warn("Alpine has already been initialized on this page. Calling Alpine.start() more than once can cause problems.");
    started = true;
    if (!document.body) warn("Unable to initialize. Trying to load Alpine before `<body>` is available. Did you forget to add `defer` in Alpine's `<script>` tag?");
    dispatch(document, "alpine:init");
    dispatch(document, "alpine:initializing");
    startObservingMutations();
    onElAdded(function (el) {
      return initTree(el, walk);
    });
    onElRemoved(function (el) {
      return destroyTree(el);
    });
    onAttributesAdded(function (el, attrs) {
      directives(el, attrs).forEach(function (handle) {
        return handle();
      });
    });
    var outNestedComponents = function outNestedComponents(el) {
      return !closestRoot(el.parentElement, true);
    };
    Array.from(document.querySelectorAll(allSelectors().join(","))).filter(outNestedComponents).forEach(function (el) {
      initTree(el);
    });
    dispatch(document, "alpine:initialized");
    setTimeout(function () {
      warnAboutMissingPlugins();
    });
  }
  var rootSelectorCallbacks = [];
  var initSelectorCallbacks = [];
  function rootSelectors() {
    return rootSelectorCallbacks.map(function (fn) {
      return fn();
    });
  }
  function allSelectors() {
    return rootSelectorCallbacks.concat(initSelectorCallbacks).map(function (fn) {
      return fn();
    });
  }
  function addRootSelector(selectorCallback) {
    rootSelectorCallbacks.push(selectorCallback);
  }
  function addInitSelector(selectorCallback) {
    initSelectorCallbacks.push(selectorCallback);
  }
  function closestRoot(el) {
    var includeInitSelectors = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : false;
    return findClosest(el, function (element) {
      var selectors = includeInitSelectors ? allSelectors() : rootSelectors();
      if (selectors.some(function (selector) {
        return element.matches(selector);
      })) return true;
    });
  }
  function findClosest(el, callback) {
    if (!el) return;
    if (callback(el)) return el;
    if (el._x_teleportBack) el = el._x_teleportBack;
    if (!el.parentElement) return;
    return findClosest(el.parentElement, callback);
  }
  function isRoot(el) {
    return rootSelectors().some(function (selector) {
      return el.matches(selector);
    });
  }
  var initInterceptors2 = [];
  function interceptInit(callback) {
    initInterceptors2.push(callback);
  }
  function initTree(el) {
    var walker = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : walk;
    var intercept = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : function () {};
    deferHandlingDirectives(function () {
      walker(el, function (el2, skip) {
        intercept(el2, skip);
        initInterceptors2.forEach(function (i) {
          return i(el2, skip);
        });
        directives(el2, el2.attributes).forEach(function (handle) {
          return handle();
        });
        el2._x_ignore && skip();
      });
    });
  }
  function destroyTree(root) {
    var walker = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : walk;
    walker(root, function (el) {
      cleanupElement(el);
      cleanupAttributes(el);
    });
  }
  function warnAboutMissingPlugins() {
    var pluginDirectives = [["ui", "dialog", ["[x-dialog], [x-popover]"]], ["anchor", "anchor", ["[x-anchor]"]], ["sort", "sort", ["[x-sort]"]]];
    pluginDirectives.forEach(function (_ref20) {
      var _ref21 = _slicedToArray(_ref20, 3),
        plugin2 = _ref21[0],
        directive2 = _ref21[1],
        selectors = _ref21[2];
      if (directiveExists(directive2)) return;
      selectors.some(function (selector) {
        if (document.querySelector(selector)) {
          warn("found \"".concat(selector, "\", but missing ").concat(plugin2, " plugin"));
          return true;
        }
      });
    });
  }

  // packages/alpinejs/src/nextTick.js
  var tickStack = [];
  var isHolding = false;
  function nextTick() {
    var callback = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : function () {};
    queueMicrotask(function () {
      isHolding || setTimeout(function () {
        releaseNextTicks();
      });
    });
    return new Promise(function (res) {
      tickStack.push(function () {
        callback();
        res();
      });
    });
  }
  function releaseNextTicks() {
    isHolding = false;
    while (tickStack.length) tickStack.shift()();
  }
  function holdNextTicks() {
    isHolding = true;
  }

  // packages/alpinejs/src/utils/classes.js
  function setClasses(el, value) {
    if (Array.isArray(value)) {
      return setClassesFromString(el, value.join(" "));
    } else if (_typeof(value) === "object" && value !== null) {
      return setClassesFromObject(el, value);
    } else if (typeof value === "function") {
      return setClasses(el, value());
    }
    return setClassesFromString(el, value);
  }
  function setClassesFromString(el, classString) {
    var split = function split(classString2) {
      return classString2.split(" ").filter(Boolean);
    };
    var missingClasses = function missingClasses(classString2) {
      return classString2.split(" ").filter(function (i) {
        return !el.classList.contains(i);
      }).filter(Boolean);
    };
    var addClassesAndReturnUndo = function addClassesAndReturnUndo(classes) {
      var _el$classList;
      (_el$classList = el.classList).add.apply(_el$classList, _toConsumableArray(classes));
      return function () {
        var _el$classList2;
        (_el$classList2 = el.classList).remove.apply(_el$classList2, _toConsumableArray(classes));
      };
    };
    classString = classString === true ? classString = "" : classString || "";
    return addClassesAndReturnUndo(missingClasses(classString));
  }
  function setClassesFromObject(el, classObject) {
    var split = function split(classString) {
      return classString.split(" ").filter(Boolean);
    };
    var forAdd = Object.entries(classObject).flatMap(function (_ref22) {
      var _ref23 = _slicedToArray(_ref22, 2),
        classString = _ref23[0],
        bool = _ref23[1];
      return bool ? split(classString) : false;
    }).filter(Boolean);
    var forRemove = Object.entries(classObject).flatMap(function (_ref24) {
      var _ref25 = _slicedToArray(_ref24, 2),
        classString = _ref25[0],
        bool = _ref25[1];
      return !bool ? split(classString) : false;
    }).filter(Boolean);
    var added = [];
    var removed = [];
    forRemove.forEach(function (i) {
      if (el.classList.contains(i)) {
        el.classList.remove(i);
        removed.push(i);
      }
    });
    forAdd.forEach(function (i) {
      if (!el.classList.contains(i)) {
        el.classList.add(i);
        added.push(i);
      }
    });
    return function () {
      removed.forEach(function (i) {
        return el.classList.add(i);
      });
      added.forEach(function (i) {
        return el.classList.remove(i);
      });
    };
  }

  // packages/alpinejs/src/utils/styles.js
  function setStyles(el, value) {
    if (_typeof(value) === "object" && value !== null) {
      return setStylesFromObject(el, value);
    }
    return setStylesFromString(el, value);
  }
  function setStylesFromObject(el, value) {
    var previousStyles = {};
    Object.entries(value).forEach(function (_ref26) {
      var _ref27 = _slicedToArray(_ref26, 2),
        key = _ref27[0],
        value2 = _ref27[1];
      previousStyles[key] = el.style[key];
      if (!key.startsWith("--")) {
        key = kebabCase(key);
      }
      el.style.setProperty(key, value2);
    });
    setTimeout(function () {
      if (el.style.length === 0) {
        el.removeAttribute("style");
      }
    });
    return function () {
      setStyles(el, previousStyles);
    };
  }
  function setStylesFromString(el, value) {
    var cache = el.getAttribute("style", value);
    el.setAttribute("style", value);
    return function () {
      el.setAttribute("style", cache || "");
    };
  }
  function kebabCase(subject) {
    return subject.replace(/([a-z])([A-Z])/g, "$1-$2").toLowerCase();
  }

  // packages/alpinejs/src/utils/once.js
  function once(callback) {
    var fallback = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : function () {};
    var called = false;
    return function () {
      if (!called) {
        called = true;
        callback.apply(this, arguments);
      } else {
        fallback.apply(this, arguments);
      }
    };
  }

  // packages/alpinejs/src/directives/x-transition.js
  directive("transition", function (el, _ref28, _ref29) {
    var value = _ref28.value,
      modifiers = _ref28.modifiers,
      expression = _ref28.expression;
    var evaluate2 = _ref29.evaluate;
    if (typeof expression === "function") expression = evaluate2(expression);
    if (expression === false) return;
    if (!expression || typeof expression === "boolean") {
      registerTransitionsFromHelper(el, modifiers, value);
    } else {
      registerTransitionsFromClassString(el, expression, value);
    }
  });
  function registerTransitionsFromClassString(el, classString, stage) {
    registerTransitionObject(el, setClasses, "");
    var directiveStorageMap = {
      "enter": function enter(classes) {
        el._x_transition.enter.during = classes;
      },
      "enter-start": function enterStart(classes) {
        el._x_transition.enter.start = classes;
      },
      "enter-end": function enterEnd(classes) {
        el._x_transition.enter.end = classes;
      },
      "leave": function leave(classes) {
        el._x_transition.leave.during = classes;
      },
      "leave-start": function leaveStart(classes) {
        el._x_transition.leave.start = classes;
      },
      "leave-end": function leaveEnd(classes) {
        el._x_transition.leave.end = classes;
      }
    };
    directiveStorageMap[stage](classString);
  }
  function registerTransitionsFromHelper(el, modifiers, stage) {
    registerTransitionObject(el, setStyles);
    var doesntSpecify = !modifiers.includes("in") && !modifiers.includes("out") && !stage;
    var transitioningIn = doesntSpecify || modifiers.includes("in") || ["enter"].includes(stage);
    var transitioningOut = doesntSpecify || modifiers.includes("out") || ["leave"].includes(stage);
    if (modifiers.includes("in") && !doesntSpecify) {
      modifiers = modifiers.filter(function (i, index) {
        return index < modifiers.indexOf("out");
      });
    }
    if (modifiers.includes("out") && !doesntSpecify) {
      modifiers = modifiers.filter(function (i, index) {
        return index > modifiers.indexOf("out");
      });
    }
    var wantsAll = !modifiers.includes("opacity") && !modifiers.includes("scale");
    var wantsOpacity = wantsAll || modifiers.includes("opacity");
    var wantsScale = wantsAll || modifiers.includes("scale");
    var opacityValue = wantsOpacity ? 0 : 1;
    var scaleValue = wantsScale ? modifierValue(modifiers, "scale", 95) / 100 : 1;
    var delay = modifierValue(modifiers, "delay", 0) / 1e3;
    var origin = modifierValue(modifiers, "origin", "center");
    var property = "opacity, transform";
    var durationIn = modifierValue(modifiers, "duration", 150) / 1e3;
    var durationOut = modifierValue(modifiers, "duration", 75) / 1e3;
    var easing = "cubic-bezier(0.4, 0.0, 0.2, 1)";
    if (transitioningIn) {
      el._x_transition.enter.during = {
        transformOrigin: origin,
        transitionDelay: "".concat(delay, "s"),
        transitionProperty: property,
        transitionDuration: "".concat(durationIn, "s"),
        transitionTimingFunction: easing
      };
      el._x_transition.enter.start = {
        opacity: opacityValue,
        transform: "scale(".concat(scaleValue, ")")
      };
      el._x_transition.enter.end = {
        opacity: 1,
        transform: "scale(1)"
      };
    }
    if (transitioningOut) {
      el._x_transition.leave.during = {
        transformOrigin: origin,
        transitionDelay: "".concat(delay, "s"),
        transitionProperty: property,
        transitionDuration: "".concat(durationOut, "s"),
        transitionTimingFunction: easing
      };
      el._x_transition.leave.start = {
        opacity: 1,
        transform: "scale(1)"
      };
      el._x_transition.leave.end = {
        opacity: opacityValue,
        transform: "scale(".concat(scaleValue, ")")
      };
    }
  }
  function registerTransitionObject(el, setFunction) {
    var defaultValue = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : {};
    if (!el._x_transition) el._x_transition = {
      enter: {
        during: defaultValue,
        start: defaultValue,
        end: defaultValue
      },
      leave: {
        during: defaultValue,
        start: defaultValue,
        end: defaultValue
      },
      "in": function _in() {
        var before = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : function () {};
        var after = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : function () {};
        transition(el, setFunction, {
          during: this.enter.during,
          start: this.enter.start,
          end: this.enter.end
        }, before, after);
      },
      out: function out() {
        var before = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : function () {};
        var after = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : function () {};
        transition(el, setFunction, {
          during: this.leave.during,
          start: this.leave.start,
          end: this.leave.end
        }, before, after);
      }
    };
  }
  window.Element.prototype._x_toggleAndCascadeWithTransitions = function (el, value, show, hide) {
    var nextTick2 = document.visibilityState === "visible" ? requestAnimationFrame : setTimeout;
    var clickAwayCompatibleShow = function clickAwayCompatibleShow() {
      return nextTick2(show);
    };
    if (value) {
      if (el._x_transition && (el._x_transition.enter || el._x_transition.leave)) {
        el._x_transition.enter && (Object.entries(el._x_transition.enter.during).length || Object.entries(el._x_transition.enter.start).length || Object.entries(el._x_transition.enter.end).length) ? el._x_transition["in"](show) : clickAwayCompatibleShow();
      } else {
        el._x_transition ? el._x_transition["in"](show) : clickAwayCompatibleShow();
      }
      return;
    }
    el._x_hidePromise = el._x_transition ? new Promise(function (resolve, reject) {
      el._x_transition.out(function () {}, function () {
        return resolve(hide);
      });
      el._x_transitioning && el._x_transitioning.beforeCancel(function () {
        return reject({
          isFromCancelledTransition: true
        });
      });
    }) : Promise.resolve(hide);
    queueMicrotask(function () {
      var closest = closestHide(el);
      if (closest) {
        if (!closest._x_hideChildren) closest._x_hideChildren = [];
        closest._x_hideChildren.push(el);
      } else {
        nextTick2(function () {
          var hideAfterChildren = function hideAfterChildren(el2) {
            var carry = Promise.all([el2._x_hidePromise].concat(_toConsumableArray((el2._x_hideChildren || []).map(hideAfterChildren)))).then(function (_ref30) {
              var _ref31 = _slicedToArray(_ref30, 1),
                i = _ref31[0];
              return i === null || i === void 0 ? void 0 : i();
            });
            delete el2._x_hidePromise;
            delete el2._x_hideChildren;
            return carry;
          };
          hideAfterChildren(el)["catch"](function (e) {
            if (!e.isFromCancelledTransition) throw e;
          });
        });
      }
    });
  };
  function closestHide(el) {
    var parent = el.parentNode;
    if (!parent) return;
    return parent._x_hidePromise ? parent : closestHide(parent);
  }
  function transition(el, setFunction) {
    var _ref32 = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : {},
      _during = _ref32.during,
      start2 = _ref32.start,
      _end = _ref32.end;
    var before = arguments.length > 3 && arguments[3] !== undefined ? arguments[3] : function () {};
    var after = arguments.length > 4 && arguments[4] !== undefined ? arguments[4] : function () {};
    if (el._x_transitioning) el._x_transitioning.cancel();
    if (Object.keys(_during).length === 0 && Object.keys(start2).length === 0 && Object.keys(_end).length === 0) {
      before();
      after();
      return;
    }
    var undoStart, undoDuring, undoEnd;
    performTransition(el, {
      start: function start() {
        undoStart = setFunction(el, start2);
      },
      during: function during() {
        undoDuring = setFunction(el, _during);
      },
      before: before,
      end: function end() {
        undoStart();
        undoEnd = setFunction(el, _end);
      },
      after: after,
      cleanup: function cleanup() {
        undoDuring();
        undoEnd();
      }
    });
  }
  function performTransition(el, stages) {
    var interrupted, reachedBefore, reachedEnd;
    var finish = once(function () {
      mutateDom(function () {
        interrupted = true;
        if (!reachedBefore) stages.before();
        if (!reachedEnd) {
          stages.end();
          releaseNextTicks();
        }
        stages.after();
        if (el.isConnected) stages.cleanup();
        delete el._x_transitioning;
      });
    });
    el._x_transitioning = {
      beforeCancels: [],
      beforeCancel: function beforeCancel(callback) {
        this.beforeCancels.push(callback);
      },
      cancel: once(function () {
        while (this.beforeCancels.length) {
          this.beforeCancels.shift()();
        }
        ;
        finish();
      }),
      finish: finish
    };
    mutateDom(function () {
      stages.start();
      stages.during();
    });
    holdNextTicks();
    requestAnimationFrame(function () {
      if (interrupted) return;
      var duration = Number(getComputedStyle(el).transitionDuration.replace(/,.*/, "").replace("s", "")) * 1e3;
      var delay = Number(getComputedStyle(el).transitionDelay.replace(/,.*/, "").replace("s", "")) * 1e3;
      if (duration === 0) duration = Number(getComputedStyle(el).animationDuration.replace("s", "")) * 1e3;
      mutateDom(function () {
        stages.before();
      });
      reachedBefore = true;
      requestAnimationFrame(function () {
        if (interrupted) return;
        mutateDom(function () {
          stages.end();
        });
        releaseNextTicks();
        setTimeout(el._x_transitioning.finish, duration + delay);
        reachedEnd = true;
      });
    });
  }
  function modifierValue(modifiers, key, fallback) {
    if (modifiers.indexOf(key) === -1) return fallback;
    var rawValue = modifiers[modifiers.indexOf(key) + 1];
    if (!rawValue) return fallback;
    if (key === "scale") {
      if (isNaN(rawValue)) return fallback;
    }
    if (key === "duration" || key === "delay") {
      var match = rawValue.match(/([0-9]+)ms/);
      if (match) return match[1];
    }
    if (key === "origin") {
      if (["top", "right", "left", "center", "bottom"].includes(modifiers[modifiers.indexOf(key) + 2])) {
        return [rawValue, modifiers[modifiers.indexOf(key) + 2]].join(" ");
      }
    }
    return rawValue;
  }

  // packages/alpinejs/src/clone.js
  var isCloning = false;
  function skipDuringClone(callback) {
    var fallback = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : function () {};
    return function () {
      return isCloning ? fallback.apply(void 0, arguments) : callback.apply(void 0, arguments);
    };
  }
  function onlyDuringClone(callback) {
    return function () {
      return isCloning && callback.apply(void 0, arguments);
    };
  }
  var interceptors = [];
  function interceptClone(callback) {
    interceptors.push(callback);
  }
  function cloneNode(from, to) {
    interceptors.forEach(function (i) {
      return i(from, to);
    });
    isCloning = true;
    dontRegisterReactiveSideEffects(function () {
      initTree(to, function (el, callback) {
        callback(el, function () {});
      });
    });
    isCloning = false;
  }
  var isCloningLegacy = false;
  function clone(oldEl, newEl) {
    if (!newEl._x_dataStack) newEl._x_dataStack = oldEl._x_dataStack;
    isCloning = true;
    isCloningLegacy = true;
    dontRegisterReactiveSideEffects(function () {
      cloneTree(newEl);
    });
    isCloning = false;
    isCloningLegacy = false;
  }
  function cloneTree(el) {
    var hasRunThroughFirstEl = false;
    var shallowWalker = function shallowWalker(el2, callback) {
      walk(el2, function (el3, skip) {
        if (hasRunThroughFirstEl && isRoot(el3)) return skip();
        hasRunThroughFirstEl = true;
        callback(el3, skip);
      });
    };
    initTree(el, shallowWalker);
  }
  function dontRegisterReactiveSideEffects(callback) {
    var cache = effect;
    overrideEffect(function (callback2, el) {
      var storedEffect = cache(callback2);
      release(storedEffect);
      return function () {};
    });
    callback();
    overrideEffect(cache);
  }

  // packages/alpinejs/src/utils/bind.js
  function bind(el, name, value) {
    var modifiers = arguments.length > 3 && arguments[3] !== undefined ? arguments[3] : [];
    if (!el._x_bindings) el._x_bindings = reactive({});
    el._x_bindings[name] = value;
    name = modifiers.includes("camel") ? camelCase(name) : name;
    switch (name) {
      case "value":
        bindInputValue(el, value);
        break;
      case "style":
        bindStyles(el, value);
        break;
      case "class":
        bindClasses(el, value);
        break;
      case "selected":
      case "checked":
        bindAttributeAndProperty(el, name, value);
        break;
      default:
        bindAttribute(el, name, value);
        break;
    }
  }
  function bindInputValue(el, value) {
    if (isRadio(el)) {
      if (el.attributes.value === void 0) {
        el.value = value;
      }
      if (window.fromModel) {
        if (typeof value === "boolean") {
          el.checked = safeParseBoolean(el.value) === value;
        } else {
          el.checked = checkedAttrLooseCompare(el.value, value);
        }
      }
    } else if (isCheckbox(el)) {
      if (Number.isInteger(value)) {
        el.value = value;
      } else if (!Array.isArray(value) && typeof value !== "boolean" && ![null, void 0].includes(value)) {
        el.value = String(value);
      } else {
        if (Array.isArray(value)) {
          el.checked = value.some(function (val) {
            return checkedAttrLooseCompare(val, el.value);
          });
        } else {
          el.checked = !!value;
        }
      }
    } else if (el.tagName === "SELECT") {
      updateSelect(el, value);
    } else {
      if (el.value === value) return;
      el.value = value === void 0 ? "" : value;
    }
  }
  function bindClasses(el, value) {
    if (el._x_undoAddedClasses) el._x_undoAddedClasses();
    el._x_undoAddedClasses = setClasses(el, value);
  }
  function bindStyles(el, value) {
    if (el._x_undoAddedStyles) el._x_undoAddedStyles();
    el._x_undoAddedStyles = setStyles(el, value);
  }
  function bindAttributeAndProperty(el, name, value) {
    bindAttribute(el, name, value);
    setPropertyIfChanged(el, name, value);
  }
  function bindAttribute(el, name, value) {
    if ([null, void 0, false].includes(value) && attributeShouldntBePreservedIfFalsy(name)) {
      el.removeAttribute(name);
    } else {
      if (isBooleanAttr(name)) value = name;
      setIfChanged(el, name, value);
    }
  }
  function setIfChanged(el, attrName, value) {
    if (el.getAttribute(attrName) != value) {
      el.setAttribute(attrName, value);
    }
  }
  function setPropertyIfChanged(el, propName, value) {
    if (el[propName] !== value) {
      el[propName] = value;
    }
  }
  function updateSelect(el, value) {
    var arrayWrappedValue = [].concat(value).map(function (value2) {
      return value2 + "";
    });
    Array.from(el.options).forEach(function (option) {
      option.selected = arrayWrappedValue.includes(option.value);
    });
  }
  function camelCase(subject) {
    return subject.toLowerCase().replace(/-(\w)/g, function (match, _char) {
      return _char.toUpperCase();
    });
  }
  function checkedAttrLooseCompare(valueA, valueB) {
    return valueA == valueB;
  }
  function safeParseBoolean(rawValue) {
    if ([1, "1", "true", "on", "yes", true].includes(rawValue)) {
      return true;
    }
    if ([0, "0", "false", "off", "no", false].includes(rawValue)) {
      return false;
    }
    return rawValue ? Boolean(rawValue) : null;
  }
  var booleanAttributes = /* @__PURE__ */new Set(["allowfullscreen", "async", "autofocus", "autoplay", "checked", "controls", "default", "defer", "disabled", "formnovalidate", "inert", "ismap", "itemscope", "loop", "multiple", "muted", "nomodule", "novalidate", "open", "playsinline", "readonly", "required", "reversed", "selected", "shadowrootclonable", "shadowrootdelegatesfocus", "shadowrootserializable"]);
  function isBooleanAttr(attrName) {
    return booleanAttributes.has(attrName);
  }
  function attributeShouldntBePreservedIfFalsy(name) {
    return !["aria-pressed", "aria-checked", "aria-expanded", "aria-selected"].includes(name);
  }
  function getBinding(el, name, fallback) {
    if (el._x_bindings && el._x_bindings[name] !== void 0) return el._x_bindings[name];
    return getAttributeBinding(el, name, fallback);
  }
  function extractProp(el, name, fallback) {
    var extract = arguments.length > 3 && arguments[3] !== undefined ? arguments[3] : true;
    if (el._x_bindings && el._x_bindings[name] !== void 0) return el._x_bindings[name];
    if (el._x_inlineBindings && el._x_inlineBindings[name] !== void 0) {
      var binding = el._x_inlineBindings[name];
      binding.extract = extract;
      return dontAutoEvaluateFunctions(function () {
        return evaluate(el, binding.expression);
      });
    }
    return getAttributeBinding(el, name, fallback);
  }
  function getAttributeBinding(el, name, fallback) {
    var attr = el.getAttribute(name);
    if (attr === null) return typeof fallback === "function" ? fallback() : fallback;
    if (attr === "") return true;
    if (isBooleanAttr(name)) {
      return !![name, "true"].includes(attr);
    }
    return attr;
  }
  function isCheckbox(el) {
    return el.type === "checkbox" || el.localName === "ui-checkbox" || el.localName === "ui-switch";
  }
  function isRadio(el) {
    return el.type === "radio" || el.localName === "ui-radio";
  }

  // packages/alpinejs/src/utils/debounce.js
  function debounce(func, wait) {
    var timeout;
    return function () {
      var context = this,
        args = arguments;
      var later = function later() {
        timeout = null;
        func.apply(context, args);
      };
      clearTimeout(timeout);
      timeout = setTimeout(later, wait);
    };
  }

  // packages/alpinejs/src/utils/throttle.js
  function throttle(func, limit) {
    var inThrottle;
    return function () {
      var context = this,
        args = arguments;
      if (!inThrottle) {
        func.apply(context, args);
        inThrottle = true;
        setTimeout(function () {
          return inThrottle = false;
        }, limit);
      }
    };
  }

  // packages/alpinejs/src/entangle.js
  function entangle(_ref33, _ref34) {
    var outerGet = _ref33.get,
      outerSet = _ref33.set;
    var innerGet = _ref34.get,
      innerSet = _ref34.set;
    var firstRun = true;
    var outerHash;
    var innerHash;
    var reference = effect(function () {
      var outer = outerGet();
      var inner = innerGet();
      if (firstRun) {
        innerSet(cloneIfObject(outer));
        firstRun = false;
      } else {
        var outerHashLatest = JSON.stringify(outer);
        var innerHashLatest = JSON.stringify(inner);
        if (outerHashLatest !== outerHash) {
          innerSet(cloneIfObject(outer));
        } else if (outerHashLatest !== innerHashLatest) {
          outerSet(cloneIfObject(inner));
        } else {}
      }
      outerHash = JSON.stringify(outerGet());
      innerHash = JSON.stringify(innerGet());
    });
    return function () {
      release(reference);
    };
  }
  function cloneIfObject(value) {
    return _typeof(value) === "object" ? JSON.parse(JSON.stringify(value)) : value;
  }

  // packages/alpinejs/src/plugin.js
  function plugin(callback) {
    var callbacks = Array.isArray(callback) ? callback : [callback];
    callbacks.forEach(function (i) {
      return i(alpine_default);
    });
  }

  // packages/alpinejs/src/store.js
  var stores = {};
  var isReactive = false;
  function store(name, value) {
    if (!isReactive) {
      stores = reactive(stores);
      isReactive = true;
    }
    if (value === void 0) {
      return stores[name];
    }
    stores[name] = value;
    initInterceptors(stores[name]);
    if (_typeof(value) === "object" && value !== null && value.hasOwnProperty("init") && typeof value.init === "function") {
      stores[name].init();
    }
  }
  function getStores() {
    return stores;
  }

  // packages/alpinejs/src/binds.js
  var binds = {};
  function bind2(name, bindings) {
    var getBindings = typeof bindings !== "function" ? function () {
      return bindings;
    } : bindings;
    if (name instanceof Element) {
      return applyBindingsObject(name, getBindings());
    } else {
      binds[name] = getBindings;
    }
    return function () {};
  }
  function injectBindingProviders(obj) {
    Object.entries(binds).forEach(function (_ref35) {
      var _ref36 = _slicedToArray(_ref35, 2),
        name = _ref36[0],
        callback = _ref36[1];
      Object.defineProperty(obj, name, {
        get: function get() {
          return function () {
            return callback.apply(void 0, arguments);
          };
        }
      });
    });
    return obj;
  }
  function applyBindingsObject(el, obj, original) {
    var cleanupRunners = [];
    while (cleanupRunners.length) cleanupRunners.pop()();
    var attributes = Object.entries(obj).map(function (_ref37) {
      var _ref38 = _slicedToArray(_ref37, 2),
        name = _ref38[0],
        value = _ref38[1];
      return {
        name: name,
        value: value
      };
    });
    var staticAttributes = attributesOnly(attributes);
    attributes = attributes.map(function (attribute) {
      if (staticAttributes.find(function (attr) {
        return attr.name === attribute.name;
      })) {
        return {
          name: "x-bind:".concat(attribute.name),
          value: "\"".concat(attribute.value, "\"")
        };
      }
      return attribute;
    });
    directives(el, attributes, original).map(function (handle) {
      cleanupRunners.push(handle.runCleanups);
      handle();
    });
    return function () {
      while (cleanupRunners.length) cleanupRunners.pop()();
    };
  }

  // packages/alpinejs/src/datas.js
  var datas = {};
  function data(name, callback) {
    datas[name] = callback;
  }
  function injectDataProviders(obj, context) {
    Object.entries(datas).forEach(function (_ref39) {
      var _ref40 = _slicedToArray(_ref39, 2),
        name = _ref40[0],
        callback = _ref40[1];
      Object.defineProperty(obj, name, {
        get: function get() {
          return function () {
            return callback.bind(context).apply(void 0, arguments);
          };
        },
        enumerable: false
      });
    });
    return obj;
  }

  // packages/alpinejs/src/alpine.js
  var Alpine = {
    get reactive() {
      return reactive;
    },
    get release() {
      return release;
    },
    get effect() {
      return effect;
    },
    get raw() {
      return raw;
    },
    version: "3.14.3",
    flushAndStopDeferringMutations: flushAndStopDeferringMutations,
    dontAutoEvaluateFunctions: dontAutoEvaluateFunctions,
    disableEffectScheduling: disableEffectScheduling,
    startObservingMutations: startObservingMutations,
    stopObservingMutations: stopObservingMutations,
    setReactivityEngine: setReactivityEngine,
    onAttributeRemoved: onAttributeRemoved,
    onAttributesAdded: onAttributesAdded,
    closestDataStack: closestDataStack,
    skipDuringClone: skipDuringClone,
    onlyDuringClone: onlyDuringClone,
    addRootSelector: addRootSelector,
    addInitSelector: addInitSelector,
    interceptClone: interceptClone,
    addScopeToNode: addScopeToNode,
    deferMutations: deferMutations,
    mapAttributes: mapAttributes,
    evaluateLater: evaluateLater,
    interceptInit: interceptInit,
    setEvaluator: setEvaluator,
    mergeProxies: mergeProxies,
    extractProp: extractProp,
    findClosest: findClosest,
    onElRemoved: onElRemoved,
    closestRoot: closestRoot,
    destroyTree: destroyTree,
    interceptor: interceptor,
    // INTERNAL: not public API and is subject to change without major release.
    transition: transition,
    // INTERNAL
    setStyles: setStyles,
    // INTERNAL
    mutateDom: mutateDom,
    directive: directive,
    entangle: entangle,
    throttle: throttle,
    debounce: debounce,
    evaluate: evaluate,
    initTree: initTree,
    nextTick: nextTick,
    prefixed: prefix,
    prefix: setPrefix,
    plugin: plugin,
    magic: magic,
    store: store,
    start: start,
    clone: clone,
    // INTERNAL
    cloneNode: cloneNode,
    // INTERNAL
    bound: getBinding,
    $data: scope,
    watch: watch,
    walk: walk,
    data: data,
    bind: bind2
  };
  var alpine_default = Alpine;

  // node_modules/@vue/shared/dist/shared.esm-bundler.js
  function makeMap(str, expectsLowerCase) {
    var map = /* @__PURE__ */Object.create(null);
    var list = str.split(",");
    for (var i = 0; i < list.length; i++) {
      map[list[i]] = true;
    }
    return expectsLowerCase ? function (val) {
      return !!map[val.toLowerCase()];
    } : function (val) {
      return !!map[val];
    };
  }
  var specialBooleanAttrs = "itemscope,allowfullscreen,formnovalidate,ismap,nomodule,novalidate,readonly";
  var isBooleanAttr2 = /* @__PURE__ */makeMap(specialBooleanAttrs + ",async,autofocus,autoplay,controls,default,defer,disabled,hidden,loop,open,required,reversed,scoped,seamless,checked,muted,multiple,selected");
  var EMPTY_OBJ = true ? Object.freeze({}) : {};
  var EMPTY_ARR = true ? Object.freeze([]) : [];
  var hasOwnProperty = Object.prototype.hasOwnProperty;
  var hasOwn = function hasOwn(val, key) {
    return hasOwnProperty.call(val, key);
  };
  var isArray = Array.isArray;
  var isMap = function isMap(val) {
    return toTypeString(val) === "[object Map]";
  };
  var isString = function isString(val) {
    return typeof val === "string";
  };
  var isSymbol = function isSymbol(val) {
    return _typeof(val) === "symbol";
  };
  var isObject = function isObject(val) {
    return val !== null && _typeof(val) === "object";
  };
  var objectToString = Object.prototype.toString;
  var toTypeString = function toTypeString(value) {
    return objectToString.call(value);
  };
  var toRawType = function toRawType(value) {
    return toTypeString(value).slice(8, -1);
  };
  var isIntegerKey = function isIntegerKey(key) {
    return isString(key) && key !== "NaN" && key[0] !== "-" && "" + parseInt(key, 10) === key;
  };
  var cacheStringFunction = function cacheStringFunction(fn) {
    var cache = /* @__PURE__ */Object.create(null);
    return function (str) {
      var hit = cache[str];
      return hit || (cache[str] = fn(str));
    };
  };
  var camelizeRE = /-(\w)/g;
  var camelize = cacheStringFunction(function (str) {
    return str.replace(camelizeRE, function (_, c) {
      return c ? c.toUpperCase() : "";
    });
  });
  var hyphenateRE = /\B([A-Z])/g;
  var hyphenate = cacheStringFunction(function (str) {
    return str.replace(hyphenateRE, "-$1").toLowerCase();
  });
  var capitalize = cacheStringFunction(function (str) {
    return str.charAt(0).toUpperCase() + str.slice(1);
  });
  var toHandlerKey = cacheStringFunction(function (str) {
    return str ? "on".concat(capitalize(str)) : "";
  });
  var hasChanged = function hasChanged(value, oldValue) {
    return value !== oldValue && (value === value || oldValue === oldValue);
  };

  // node_modules/@vue/reactivity/dist/reactivity.esm-bundler.js
  var targetMap = /* @__PURE__ */new WeakMap();
  var effectStack = [];
  var activeEffect;
  var ITERATE_KEY = Symbol(true ? "iterate" : "");
  var MAP_KEY_ITERATE_KEY = Symbol(true ? "Map key iterate" : "");
  function isEffect(fn) {
    return fn && fn._isEffect === true;
  }
  function effect2(fn) {
    var options = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : EMPTY_OBJ;
    if (isEffect(fn)) {
      fn = fn.raw;
    }
    var effect3 = createReactiveEffect(fn, options);
    if (!options.lazy) {
      effect3();
    }
    return effect3;
  }
  function stop(effect3) {
    if (effect3.active) {
      cleanup(effect3);
      if (effect3.options.onStop) {
        effect3.options.onStop();
      }
      effect3.active = false;
    }
  }
  var uid = 0;
  function createReactiveEffect(fn, options) {
    var effect3 = function reactiveEffect() {
      if (!effect3.active) {
        return fn();
      }
      if (!effectStack.includes(effect3)) {
        cleanup(effect3);
        try {
          enableTracking();
          effectStack.push(effect3);
          activeEffect = effect3;
          return fn();
        } finally {
          effectStack.pop();
          resetTracking();
          activeEffect = effectStack[effectStack.length - 1];
        }
      }
    };
    effect3.id = uid++;
    effect3.allowRecurse = !!options.allowRecurse;
    effect3._isEffect = true;
    effect3.active = true;
    effect3.raw = fn;
    effect3.deps = [];
    effect3.options = options;
    return effect3;
  }
  function cleanup(effect3) {
    var deps = effect3.deps;
    if (deps.length) {
      for (var i = 0; i < deps.length; i++) {
        deps[i]["delete"](effect3);
      }
      deps.length = 0;
    }
  }
  var shouldTrack = true;
  var trackStack = [];
  function pauseTracking() {
    trackStack.push(shouldTrack);
    shouldTrack = false;
  }
  function enableTracking() {
    trackStack.push(shouldTrack);
    shouldTrack = true;
  }
  function resetTracking() {
    var last = trackStack.pop();
    shouldTrack = last === void 0 ? true : last;
  }
  function track(target, type, key) {
    if (!shouldTrack || activeEffect === void 0) {
      return;
    }
    var depsMap = targetMap.get(target);
    if (!depsMap) {
      targetMap.set(target, depsMap = /* @__PURE__ */new Map());
    }
    var dep = depsMap.get(key);
    if (!dep) {
      depsMap.set(key, dep = /* @__PURE__ */new Set());
    }
    if (!dep.has(activeEffect)) {
      dep.add(activeEffect);
      activeEffect.deps.push(dep);
      if (activeEffect.options.onTrack) {
        activeEffect.options.onTrack({
          effect: activeEffect,
          target: target,
          type: type,
          key: key
        });
      }
    }
  }
  function trigger(target, type, key, newValue, oldValue, oldTarget) {
    var depsMap = targetMap.get(target);
    if (!depsMap) {
      return;
    }
    var effects = /* @__PURE__ */new Set();
    var add2 = function add2(effectsToAdd) {
      if (effectsToAdd) {
        effectsToAdd.forEach(function (effect3) {
          if (effect3 !== activeEffect || effect3.allowRecurse) {
            effects.add(effect3);
          }
        });
      }
    };
    if (type === "clear") {
      depsMap.forEach(add2);
    } else if (key === "length" && isArray(target)) {
      depsMap.forEach(function (dep, key2) {
        if (key2 === "length" || key2 >= newValue) {
          add2(dep);
        }
      });
    } else {
      if (key !== void 0) {
        add2(depsMap.get(key));
      }
      switch (type) {
        case "add":
          if (!isArray(target)) {
            add2(depsMap.get(ITERATE_KEY));
            if (isMap(target)) {
              add2(depsMap.get(MAP_KEY_ITERATE_KEY));
            }
          } else if (isIntegerKey(key)) {
            add2(depsMap.get("length"));
          }
          break;
        case "delete":
          if (!isArray(target)) {
            add2(depsMap.get(ITERATE_KEY));
            if (isMap(target)) {
              add2(depsMap.get(MAP_KEY_ITERATE_KEY));
            }
          }
          break;
        case "set":
          if (isMap(target)) {
            add2(depsMap.get(ITERATE_KEY));
          }
          break;
      }
    }
    var run = function run(effect3) {
      if (effect3.options.onTrigger) {
        effect3.options.onTrigger({
          effect: effect3,
          target: target,
          key: key,
          type: type,
          newValue: newValue,
          oldValue: oldValue,
          oldTarget: oldTarget
        });
      }
      if (effect3.options.scheduler) {
        effect3.options.scheduler(effect3);
      } else {
        effect3();
      }
    };
    effects.forEach(run);
  }
  var isNonTrackableKeys = /* @__PURE__ */makeMap("__proto__,__v_isRef,__isVue");
  var builtInSymbols = new Set(Object.getOwnPropertyNames(Symbol).map(function (key) {
    return Symbol[key];
  }).filter(isSymbol));
  var get2 = /* @__PURE__ */createGetter();
  var readonlyGet = /* @__PURE__ */createGetter(true);
  var arrayInstrumentations = /* @__PURE__ */createArrayInstrumentations();
  function createArrayInstrumentations() {
    var instrumentations = {};
    ["includes", "indexOf", "lastIndexOf"].forEach(function (key) {
      instrumentations[key] = function () {
        var arr = toRaw(this);
        for (var i = 0, l = this.length; i < l; i++) {
          track(arr, "get", i + "");
        }
        for (var _len3 = arguments.length, args = new Array(_len3), _key3 = 0; _key3 < _len3; _key3++) {
          args[_key3] = arguments[_key3];
        }
        var res = arr[key].apply(arr, args);
        if (res === -1 || res === false) {
          return arr[key].apply(arr, _toConsumableArray(args.map(toRaw)));
        } else {
          return res;
        }
      };
    });
    ["push", "pop", "shift", "unshift", "splice"].forEach(function (key) {
      instrumentations[key] = function () {
        pauseTracking();
        for (var _len4 = arguments.length, args = new Array(_len4), _key4 = 0; _key4 < _len4; _key4++) {
          args[_key4] = arguments[_key4];
        }
        var res = toRaw(this)[key].apply(this, args);
        resetTracking();
        return res;
      };
    });
    return instrumentations;
  }
  function createGetter() {
    var isReadonly = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : false;
    var shallow = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : false;
    return function get3(target, key, receiver) {
      if (key === "__v_isReactive") {
        return !isReadonly;
      } else if (key === "__v_isReadonly") {
        return isReadonly;
      } else if (key === "__v_raw" && receiver === (isReadonly ? shallow ? shallowReadonlyMap : readonlyMap : shallow ? shallowReactiveMap : reactiveMap).get(target)) {
        return target;
      }
      var targetIsArray = isArray(target);
      if (!isReadonly && targetIsArray && hasOwn(arrayInstrumentations, key)) {
        return Reflect.get(arrayInstrumentations, key, receiver);
      }
      var res = Reflect.get(target, key, receiver);
      if (isSymbol(key) ? builtInSymbols.has(key) : isNonTrackableKeys(key)) {
        return res;
      }
      if (!isReadonly) {
        track(target, "get", key);
      }
      if (shallow) {
        return res;
      }
      if (isRef(res)) {
        var shouldUnwrap = !targetIsArray || !isIntegerKey(key);
        return shouldUnwrap ? res.value : res;
      }
      if (isObject(res)) {
        return isReadonly ? readonly(res) : reactive2(res);
      }
      return res;
    };
  }
  var set2 = /* @__PURE__ */createSetter();
  function createSetter() {
    var shallow = arguments.length > 0 && arguments[0] !== undefined ? arguments[0] : false;
    return function set3(target, key, value, receiver) {
      var oldValue = target[key];
      if (!shallow) {
        value = toRaw(value);
        oldValue = toRaw(oldValue);
        if (!isArray(target) && isRef(oldValue) && !isRef(value)) {
          oldValue.value = value;
          return true;
        }
      }
      var hadKey = isArray(target) && isIntegerKey(key) ? Number(key) < target.length : hasOwn(target, key);
      var result = Reflect.set(target, key, value, receiver);
      if (target === toRaw(receiver)) {
        if (!hadKey) {
          trigger(target, "add", key, value);
        } else if (hasChanged(value, oldValue)) {
          trigger(target, "set", key, value, oldValue);
        }
      }
      return result;
    };
  }
  function deleteProperty(target, key) {
    var hadKey = hasOwn(target, key);
    var oldValue = target[key];
    var result = Reflect.deleteProperty(target, key);
    if (result && hadKey) {
      trigger(target, "delete", key, void 0, oldValue);
    }
    return result;
  }
  function has(target, key) {
    var result = Reflect.has(target, key);
    if (!isSymbol(key) || !builtInSymbols.has(key)) {
      track(target, "has", key);
    }
    return result;
  }
  function ownKeys(target) {
    track(target, "iterate", isArray(target) ? "length" : ITERATE_KEY);
    return Reflect.ownKeys(target);
  }
  var mutableHandlers = {
    get: get2,
    set: set2,
    deleteProperty: deleteProperty,
    has: has,
    ownKeys: ownKeys
  };
  var readonlyHandlers = {
    get: readonlyGet,
    set: function set(target, key) {
      if (true) {
        console.warn("Set operation on key \"".concat(String(key), "\" failed: target is readonly."), target);
      }
      return true;
    },
    deleteProperty: function deleteProperty(target, key) {
      if (true) {
        console.warn("Delete operation on key \"".concat(String(key), "\" failed: target is readonly."), target);
      }
      return true;
    }
  };
  var toReactive = function toReactive(value) {
    return isObject(value) ? reactive2(value) : value;
  };
  var toReadonly = function toReadonly(value) {
    return isObject(value) ? readonly(value) : value;
  };
  var toShallow = function toShallow(value) {
    return value;
  };
  var getProto = function getProto(v) {
    return Reflect.getPrototypeOf(v);
  };
  function get$1(target, key) {
    var isReadonly = arguments.length > 2 && arguments[2] !== undefined ? arguments[2] : false;
    var isShallow = arguments.length > 3 && arguments[3] !== undefined ? arguments[3] : false;
    target = target["__v_raw"
    /* RAW */];

    var rawTarget = toRaw(target);
    var rawKey = toRaw(key);
    if (key !== rawKey) {
      !isReadonly && track(rawTarget, "get", key);
    }
    !isReadonly && track(rawTarget, "get", rawKey);
    var _getProto = getProto(rawTarget),
      has2 = _getProto.has;
    var wrap = isShallow ? toShallow : isReadonly ? toReadonly : toReactive;
    if (has2.call(rawTarget, key)) {
      return wrap(target.get(key));
    } else if (has2.call(rawTarget, rawKey)) {
      return wrap(target.get(rawKey));
    } else if (target !== rawTarget) {
      target.get(key);
    }
  }
  function has$1(key) {
    var isReadonly = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : false;
    var target = this["__v_raw"
    /* RAW */];

    var rawTarget = toRaw(target);
    var rawKey = toRaw(key);
    if (key !== rawKey) {
      !isReadonly && track(rawTarget, "has", key);
    }
    !isReadonly && track(rawTarget, "has", rawKey);
    return key === rawKey ? target.has(key) : target.has(key) || target.has(rawKey);
  }
  function size(target) {
    var isReadonly = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : false;
    target = target["__v_raw"
    /* RAW */];

    !isReadonly && track(toRaw(target), "iterate", ITERATE_KEY);
    return Reflect.get(target, "size", target);
  }
  function add(value) {
    value = toRaw(value);
    var target = toRaw(this);
    var proto = getProto(target);
    var hadKey = proto.has.call(target, value);
    if (!hadKey) {
      target.add(value);
      trigger(target, "add", value, value);
    }
    return this;
  }
  function set$1(key, value) {
    value = toRaw(value);
    var target = toRaw(this);
    var _getProto2 = getProto(target),
      has2 = _getProto2.has,
      get3 = _getProto2.get;
    var hadKey = has2.call(target, key);
    if (!hadKey) {
      key = toRaw(key);
      hadKey = has2.call(target, key);
    } else if (true) {
      checkIdentityKeys(target, has2, key);
    }
    var oldValue = get3.call(target, key);
    target.set(key, value);
    if (!hadKey) {
      trigger(target, "add", key, value);
    } else if (hasChanged(value, oldValue)) {
      trigger(target, "set", key, value, oldValue);
    }
    return this;
  }
  function deleteEntry(key) {
    var target = toRaw(this);
    var _getProto3 = getProto(target),
      has2 = _getProto3.has,
      get3 = _getProto3.get;
    var hadKey = has2.call(target, key);
    if (!hadKey) {
      key = toRaw(key);
      hadKey = has2.call(target, key);
    } else if (true) {
      checkIdentityKeys(target, has2, key);
    }
    var oldValue = get3 ? get3.call(target, key) : void 0;
    var result = target["delete"](key);
    if (hadKey) {
      trigger(target, "delete", key, void 0, oldValue);
    }
    return result;
  }
  function clear() {
    var target = toRaw(this);
    var hadItems = target.size !== 0;
    var oldTarget = true ? isMap(target) ? new Map(target) : new Set(target) : void 0;
    var result = target.clear();
    if (hadItems) {
      trigger(target, "clear", void 0, void 0, oldTarget);
    }
    return result;
  }
  function createForEach(isReadonly, isShallow) {
    return function forEach(callback, thisArg) {
      var observed = this;
      var target = observed["__v_raw"
      /* RAW */];

      var rawTarget = toRaw(target);
      var wrap = isShallow ? toShallow : isReadonly ? toReadonly : toReactive;
      !isReadonly && track(rawTarget, "iterate", ITERATE_KEY);
      return target.forEach(function (value, key) {
        return callback.call(thisArg, wrap(value), wrap(key), observed);
      });
    };
  }
  function createIterableMethod(method, isReadonly, isShallow) {
    return function () {
      var target = this["__v_raw"
      /* RAW */];

      var rawTarget = toRaw(target);
      var targetIsMap = isMap(rawTarget);
      var isPair = method === "entries" || method === Symbol.iterator && targetIsMap;
      var isKeyOnly = method === "keys" && targetIsMap;
      var innerIterator = target[method].apply(target, arguments);
      var wrap = isShallow ? toShallow : isReadonly ? toReadonly : toReactive;
      !isReadonly && track(rawTarget, "iterate", isKeyOnly ? MAP_KEY_ITERATE_KEY : ITERATE_KEY);
      return _defineProperty({
        // iterator protocol
        next: function next() {
          var _innerIterator$next = innerIterator.next(),
            value = _innerIterator$next.value,
            done = _innerIterator$next.done;
          return done ? {
            value: value,
            done: done
          } : {
            value: isPair ? [wrap(value[0]), wrap(value[1])] : wrap(value),
            done: done
          };
        }
      }, Symbol.iterator, function () {
        return this;
      });
    };
  }
  function createReadonlyMethod(type) {
    return function () {
      if (true) {
        var key = (arguments.length <= 0 ? undefined : arguments[0]) ? "on key \"".concat(arguments.length <= 0 ? undefined : arguments[0], "\" ") : "";
        console.warn("".concat(capitalize(type), " operation ").concat(key, "failed: target is readonly."), toRaw(this));
      }
      return type === "delete" ? false : this;
    };
  }
  function createInstrumentations() {
    var mutableInstrumentations2 = {
      get: function get(key) {
        return get$1(this, key);
      },
      get size() {
        return size(this);
      },
      has: has$1,
      add: add,
      set: set$1,
      "delete": deleteEntry,
      clear: clear,
      forEach: createForEach(false, false)
    };
    var shallowInstrumentations2 = {
      get: function get(key) {
        return get$1(this, key, false, true);
      },
      get size() {
        return size(this);
      },
      has: has$1,
      add: add,
      set: set$1,
      "delete": deleteEntry,
      clear: clear,
      forEach: createForEach(false, true)
    };
    var readonlyInstrumentations2 = {
      get: function get(key) {
        return get$1(this, key, true);
      },
      get size() {
        return size(this, true);
      },
      has: function has(key) {
        return has$1.call(this, key, true);
      },
      add: createReadonlyMethod("add"
      /* ADD */),

      set: createReadonlyMethod("set"
      /* SET */),

      "delete": createReadonlyMethod("delete"
      /* DELETE */),

      clear: createReadonlyMethod("clear"
      /* CLEAR */),

      forEach: createForEach(true, false)
    };
    var shallowReadonlyInstrumentations2 = {
      get: function get(key) {
        return get$1(this, key, true, true);
      },
      get size() {
        return size(this, true);
      },
      has: function has(key) {
        return has$1.call(this, key, true);
      },
      add: createReadonlyMethod("add"
      /* ADD */),

      set: createReadonlyMethod("set"
      /* SET */),

      "delete": createReadonlyMethod("delete"
      /* DELETE */),

      clear: createReadonlyMethod("clear"
      /* CLEAR */),

      forEach: createForEach(true, true)
    };
    var iteratorMethods = ["keys", "values", "entries", Symbol.iterator];
    iteratorMethods.forEach(function (method) {
      mutableInstrumentations2[method] = createIterableMethod(method, false, false);
      readonlyInstrumentations2[method] = createIterableMethod(method, true, false);
      shallowInstrumentations2[method] = createIterableMethod(method, false, true);
      shallowReadonlyInstrumentations2[method] = createIterableMethod(method, true, true);
    });
    return [mutableInstrumentations2, readonlyInstrumentations2, shallowInstrumentations2, shallowReadonlyInstrumentations2];
  }
  var _createInstrumentatio = /* @__PURE__ */createInstrumentations(),
    _createInstrumentatio2 = _slicedToArray(_createInstrumentatio, 4),
    mutableInstrumentations = _createInstrumentatio2[0],
    readonlyInstrumentations = _createInstrumentatio2[1],
    shallowInstrumentations = _createInstrumentatio2[2],
    shallowReadonlyInstrumentations = _createInstrumentatio2[3];
  function createInstrumentationGetter(isReadonly, shallow) {
    var instrumentations = shallow ? isReadonly ? shallowReadonlyInstrumentations : shallowInstrumentations : isReadonly ? readonlyInstrumentations : mutableInstrumentations;
    return function (target, key, receiver) {
      if (key === "__v_isReactive") {
        return !isReadonly;
      } else if (key === "__v_isReadonly") {
        return isReadonly;
      } else if (key === "__v_raw") {
        return target;
      }
      return Reflect.get(hasOwn(instrumentations, key) && key in target ? instrumentations : target, key, receiver);
    };
  }
  var mutableCollectionHandlers = {
    get: /* @__PURE__ */createInstrumentationGetter(false, false)
  };
  var readonlyCollectionHandlers = {
    get: /* @__PURE__ */createInstrumentationGetter(true, false)
  };
  function checkIdentityKeys(target, has2, key) {
    var rawKey = toRaw(key);
    if (rawKey !== key && has2.call(target, rawKey)) {
      var type = toRawType(target);
      console.warn("Reactive ".concat(type, " contains both the raw and reactive versions of the same object").concat(type === "Map" ? " as keys" : "", ", which can lead to inconsistencies. Avoid differentiating between the raw and reactive versions of an object and only use the reactive version if possible."));
    }
  }
  var reactiveMap = /* @__PURE__ */new WeakMap();
  var shallowReactiveMap = /* @__PURE__ */new WeakMap();
  var readonlyMap = /* @__PURE__ */new WeakMap();
  var shallowReadonlyMap = /* @__PURE__ */new WeakMap();
  function targetTypeMap(rawType) {
    switch (rawType) {
      case "Object":
      case "Array":
        return 1;
      case "Map":
      case "Set":
      case "WeakMap":
      case "WeakSet":
        return 2;
      default:
        return 0;
    }
  }
  function getTargetType(value) {
    return value["__v_skip"
    /* SKIP */] || !Object.isExtensible(value) ? 0 : targetTypeMap(toRawType(value));
  }
  function reactive2(target) {
    if (target && target["__v_isReadonly"
    /* IS_READONLY */]) {
      return target;
    }
    return createReactiveObject(target, false, mutableHandlers, mutableCollectionHandlers, reactiveMap);
  }
  function readonly(target) {
    return createReactiveObject(target, true, readonlyHandlers, readonlyCollectionHandlers, readonlyMap);
  }
  function createReactiveObject(target, isReadonly, baseHandlers, collectionHandlers, proxyMap) {
    if (!isObject(target)) {
      if (true) {
        console.warn("value cannot be made reactive: ".concat(String(target)));
      }
      return target;
    }
    if (target["__v_raw"
    /* RAW */] && !(isReadonly && target["__v_isReactive"
    /* IS_REACTIVE */])) {
      return target;
    }
    var existingProxy = proxyMap.get(target);
    if (existingProxy) {
      return existingProxy;
    }
    var targetType = getTargetType(target);
    if (targetType === 0) {
      return target;
    }
    var proxy = new Proxy(target, targetType === 2 ? collectionHandlers : baseHandlers);
    proxyMap.set(target, proxy);
    return proxy;
  }
  function toRaw(observed) {
    return observed && toRaw(observed["__v_raw"
    /* RAW */]) || observed;
  }
  function isRef(r) {
    return Boolean(r && r.__v_isRef === true);
  }

  // packages/alpinejs/src/magics/$nextTick.js
  magic("nextTick", function () {
    return nextTick;
  });

  // packages/alpinejs/src/magics/$dispatch.js
  magic("dispatch", function (el) {
    return dispatch.bind(dispatch, el);
  });

  // packages/alpinejs/src/magics/$watch.js
  magic("watch", function (el, _ref42) {
    var evaluateLater2 = _ref42.evaluateLater,
      cleanup2 = _ref42.cleanup;
    return function (key, callback) {
      var evaluate2 = evaluateLater2(key);
      var getter = function getter() {
        var value;
        evaluate2(function (i) {
          return value = i;
        });
        return value;
      };
      var unwatch = watch(getter, callback);
      cleanup2(unwatch);
    };
  });

  // packages/alpinejs/src/magics/$store.js
  magic("store", getStores);

  // packages/alpinejs/src/magics/$data.js
  magic("data", function (el) {
    return scope(el);
  });

  // packages/alpinejs/src/magics/$root.js
  magic("root", function (el) {
    return closestRoot(el);
  });

  // packages/alpinejs/src/magics/$refs.js
  magic("refs", function (el) {
    if (el._x_refs_proxy) return el._x_refs_proxy;
    el._x_refs_proxy = mergeProxies(getArrayOfRefObject(el));
    return el._x_refs_proxy;
  });
  function getArrayOfRefObject(el) {
    var refObjects = [];
    findClosest(el, function (i) {
      if (i._x_refs) refObjects.push(i._x_refs);
    });
    return refObjects;
  }

  // packages/alpinejs/src/ids.js
  var globalIdMemo = {};
  function findAndIncrementId(name) {
    if (!globalIdMemo[name]) globalIdMemo[name] = 0;
    return ++globalIdMemo[name];
  }
  function closestIdRoot(el, name) {
    return findClosest(el, function (element) {
      if (element._x_ids && element._x_ids[name]) return true;
    });
  }
  function setIdRoot(el, name) {
    if (!el._x_ids) el._x_ids = {};
    if (!el._x_ids[name]) el._x_ids[name] = findAndIncrementId(name);
  }

  // packages/alpinejs/src/magics/$id.js
  magic("id", function (el, _ref43) {
    var cleanup2 = _ref43.cleanup;
    return function (name) {
      var key = arguments.length > 1 && arguments[1] !== undefined ? arguments[1] : null;
      var cacheKey = "".concat(name).concat(key ? "-".concat(key) : "");
      return cacheIdByNameOnElement(el, cacheKey, cleanup2, function () {
        var root = closestIdRoot(el, name);
        var id = root ? root._x_ids[name] : findAndIncrementId(name);
        return key ? "".concat(name, "-").concat(id, "-").concat(key) : "".concat(name, "-").concat(id);
      });
    };
  });
  interceptClone(function (from, to) {
    if (from._x_id) {
      to._x_id = from._x_id;
    }
  });
  function cacheIdByNameOnElement(el, cacheKey, cleanup2, callback) {
    if (!el._x_id) el._x_id = {};
    if (el._x_id[cacheKey]) return el._x_id[cacheKey];
    var output = callback();
    el._x_id[cacheKey] = output;
    cleanup2(function () {
      delete el._x_id[cacheKey];
    });
    return output;
  }

  // packages/alpinejs/src/magics/$el.js
  magic("el", function (el) {
    return el;
  });

  // packages/alpinejs/src/magics/index.js
  warnMissingPluginMagic("Focus", "focus", "focus");
  warnMissingPluginMagic("Persist", "persist", "persist");
  function warnMissingPluginMagic(name, magicName, slug) {
    magic(magicName, function (el) {
      return warn("You can't use [$".concat(magicName, "] without first installing the \"").concat(name, "\" plugin here: https://alpinejs.dev/plugins/").concat(slug), el);
    });
  }

  // packages/alpinejs/src/directives/x-modelable.js
  directive("modelable", function (el, _ref44, _ref45) {
    var expression = _ref44.expression;
    var effect3 = _ref45.effect,
      evaluateLater2 = _ref45.evaluateLater,
      cleanup2 = _ref45.cleanup;
    var func = evaluateLater2(expression);
    var innerGet = function innerGet() {
      var result;
      func(function (i) {
        return result = i;
      });
      return result;
    };
    var evaluateInnerSet = evaluateLater2("".concat(expression, " = __placeholder"));
    var innerSet = function innerSet(val) {
      return evaluateInnerSet(function () {}, {
        scope: {
          "__placeholder": val
        }
      });
    };
    var initialValue = innerGet();
    innerSet(initialValue);
    queueMicrotask(function () {
      if (!el._x_model) return;
      el._x_removeModelListeners["default"]();
      var outerGet = el._x_model.get;
      var outerSet = el._x_model.set;
      var releaseEntanglement = entangle({
        get: function get() {
          return outerGet();
        },
        set: function set(value) {
          outerSet(value);
        }
      }, {
        get: function get() {
          return innerGet();
        },
        set: function set(value) {
          innerSet(value);
        }
      });
      cleanup2(releaseEntanglement);
    });
  });

  // packages/alpinejs/src/directives/x-teleport.js
  directive("teleport", function (el, _ref46, _ref47) {
    var modifiers = _ref46.modifiers,
      expression = _ref46.expression;
    var cleanup2 = _ref47.cleanup;
    if (el.tagName.toLowerCase() !== "template") warn("x-teleport can only be used on a <template> tag", el);
    var target = getTarget(expression);
    var clone2 = el.content.cloneNode(true).firstElementChild;
    el._x_teleport = clone2;
    clone2._x_teleportBack = el;
    el.setAttribute("data-teleport-template", true);
    clone2.setAttribute("data-teleport-target", true);
    if (el._x_forwardEvents) {
      el._x_forwardEvents.forEach(function (eventName) {
        clone2.addEventListener(eventName, function (e) {
          e.stopPropagation();
          el.dispatchEvent(new e.constructor(e.type, e));
        });
      });
    }
    addScopeToNode(clone2, {}, el);
    var placeInDom = function placeInDom(clone3, target2, modifiers2) {
      if (modifiers2.includes("prepend")) {
        target2.parentNode.insertBefore(clone3, target2);
      } else if (modifiers2.includes("append")) {
        target2.parentNode.insertBefore(clone3, target2.nextSibling);
      } else {
        target2.appendChild(clone3);
      }
    };
    mutateDom(function () {
      placeInDom(clone2, target, modifiers);
      skipDuringClone(function () {
        initTree(clone2);
        clone2._x_ignore = true;
      })();
    });
    el._x_teleportPutBack = function () {
      var target2 = getTarget(expression);
      mutateDom(function () {
        placeInDom(el._x_teleport, target2, modifiers);
      });
    };
    cleanup2(function () {
      return mutateDom(function () {
        clone2.remove();
        destroyTree(clone2);
      });
    });
  });
  var teleportContainerDuringClone = document.createElement("div");
  function getTarget(expression) {
    var target = skipDuringClone(function () {
      return document.querySelector(expression);
    }, function () {
      return teleportContainerDuringClone;
    })();
    if (!target) warn("Cannot find x-teleport element for selector: \"".concat(expression, "\""));
    return target;
  }

  // packages/alpinejs/src/directives/x-ignore.js
  var handler = function handler() {};
  handler.inline = function (el, _ref48, _ref49) {
    var modifiers = _ref48.modifiers;
    var cleanup2 = _ref49.cleanup;
    modifiers.includes("self") ? el._x_ignoreSelf = true : el._x_ignore = true;
    cleanup2(function () {
      modifiers.includes("self") ? delete el._x_ignoreSelf : delete el._x_ignore;
    });
  };
  directive("ignore", handler);

  // packages/alpinejs/src/directives/x-effect.js
  directive("effect", skipDuringClone(function (el, _ref50, _ref51) {
    var expression = _ref50.expression;
    var effect3 = _ref51.effect;
    effect3(evaluateLater(el, expression));
  }));

  // packages/alpinejs/src/utils/on.js
  function on(el, event, modifiers, callback) {
    var listenerTarget = el;
    var handler4 = function handler4(e) {
      return callback(e);
    };
    var options = {};
    var wrapHandler = function wrapHandler(callback2, wrapper) {
      return function (e) {
        return wrapper(callback2, e);
      };
    };
    if (modifiers.includes("dot")) event = dotSyntax(event);
    if (modifiers.includes("camel")) event = camelCase2(event);
    if (modifiers.includes("passive")) options.passive = true;
    if (modifiers.includes("capture")) options.capture = true;
    if (modifiers.includes("window")) listenerTarget = window;
    if (modifiers.includes("document")) listenerTarget = document;
    if (modifiers.includes("debounce")) {
      var nextModifier = modifiers[modifiers.indexOf("debounce") + 1] || "invalid-wait";
      var wait = isNumeric(nextModifier.split("ms")[0]) ? Number(nextModifier.split("ms")[0]) : 250;
      handler4 = debounce(handler4, wait);
    }
    if (modifiers.includes("throttle")) {
      var _nextModifier = modifiers[modifiers.indexOf("throttle") + 1] || "invalid-wait";
      var _wait = isNumeric(_nextModifier.split("ms")[0]) ? Number(_nextModifier.split("ms")[0]) : 250;
      handler4 = throttle(handler4, _wait);
    }
    if (modifiers.includes("prevent")) handler4 = wrapHandler(handler4, function (next, e) {
      e.preventDefault();
      next(e);
    });
    if (modifiers.includes("stop")) handler4 = wrapHandler(handler4, function (next, e) {
      e.stopPropagation();
      next(e);
    });
    if (modifiers.includes("once")) {
      handler4 = wrapHandler(handler4, function (next, e) {
        next(e);
        listenerTarget.removeEventListener(event, handler4, options);
      });
    }
    if (modifiers.includes("away") || modifiers.includes("outside")) {
      listenerTarget = document;
      handler4 = wrapHandler(handler4, function (next, e) {
        if (el.contains(e.target)) return;
        if (e.target.isConnected === false) return;
        if (el.offsetWidth < 1 && el.offsetHeight < 1) return;
        if (el._x_isShown === false) return;
        next(e);
      });
    }
    if (modifiers.includes("self")) handler4 = wrapHandler(handler4, function (next, e) {
      e.target === el && next(e);
    });
    if (isKeyEvent(event) || isClickEvent(event)) {
      handler4 = wrapHandler(handler4, function (next, e) {
        if (isListeningForASpecificKeyThatHasntBeenPressed(e, modifiers)) {
          return;
        }
        next(e);
      });
    }
    listenerTarget.addEventListener(event, handler4, options);
    return function () {
      listenerTarget.removeEventListener(event, handler4, options);
    };
  }
  function dotSyntax(subject) {
    return subject.replace(/-/g, ".");
  }
  function camelCase2(subject) {
    return subject.toLowerCase().replace(/-(\w)/g, function (match, _char2) {
      return _char2.toUpperCase();
    });
  }
  function isNumeric(subject) {
    return !Array.isArray(subject) && !isNaN(subject);
  }
  function kebabCase2(subject) {
    if ([" ", "_"].includes(subject)) return subject;
    return subject.replace(/([a-z])([A-Z])/g, "$1-$2").replace(/[_\s]/, "-").toLowerCase();
  }
  function isKeyEvent(event) {
    return ["keydown", "keyup"].includes(event);
  }
  function isClickEvent(event) {
    return ["contextmenu", "click", "mouse"].some(function (i) {
      return event.includes(i);
    });
  }
  function isListeningForASpecificKeyThatHasntBeenPressed(e, modifiers) {
    var keyModifiers = modifiers.filter(function (i) {
      return !["window", "document", "prevent", "stop", "once", "capture", "self", "away", "outside", "passive"].includes(i);
    });
    if (keyModifiers.includes("debounce")) {
      var debounceIndex = keyModifiers.indexOf("debounce");
      keyModifiers.splice(debounceIndex, isNumeric((keyModifiers[debounceIndex + 1] || "invalid-wait").split("ms")[0]) ? 2 : 1);
    }
    if (keyModifiers.includes("throttle")) {
      var _debounceIndex = keyModifiers.indexOf("throttle");
      keyModifiers.splice(_debounceIndex, isNumeric((keyModifiers[_debounceIndex + 1] || "invalid-wait").split("ms")[0]) ? 2 : 1);
    }
    if (keyModifiers.length === 0) return false;
    if (keyModifiers.length === 1 && keyToModifiers(e.key).includes(keyModifiers[0])) return false;
    var systemKeyModifiers = ["ctrl", "shift", "alt", "meta", "cmd", "super"];
    var selectedSystemKeyModifiers = systemKeyModifiers.filter(function (modifier) {
      return keyModifiers.includes(modifier);
    });
    keyModifiers = keyModifiers.filter(function (i) {
      return !selectedSystemKeyModifiers.includes(i);
    });
    if (selectedSystemKeyModifiers.length > 0) {
      var activelyPressedKeyModifiers = selectedSystemKeyModifiers.filter(function (modifier) {
        if (modifier === "cmd" || modifier === "super") modifier = "meta";
        return e["".concat(modifier, "Key")];
      });
      if (activelyPressedKeyModifiers.length === selectedSystemKeyModifiers.length) {
        if (isClickEvent(e.type)) return false;
        if (keyToModifiers(e.key).includes(keyModifiers[0])) return false;
      }
    }
    return true;
  }
  function keyToModifiers(key) {
    if (!key) return [];
    key = kebabCase2(key);
    var modifierToKeyMap = {
      "ctrl": "control",
      "slash": "/",
      "space": " ",
      "spacebar": " ",
      "cmd": "meta",
      "esc": "escape",
      "up": "arrow-up",
      "down": "arrow-down",
      "left": "arrow-left",
      "right": "arrow-right",
      "period": ".",
      "comma": ",",
      "equal": "=",
      "minus": "-",
      "underscore": "_"
    };
    modifierToKeyMap[key] = key;
    return Object.keys(modifierToKeyMap).map(function (modifier) {
      if (modifierToKeyMap[modifier] === key) return modifier;
    }).filter(function (modifier) {
      return modifier;
    });
  }

  // packages/alpinejs/src/directives/x-model.js
  directive("model", function (el, _ref52, _ref53) {
    var modifiers = _ref52.modifiers,
      expression = _ref52.expression;
    var effect3 = _ref53.effect,
      cleanup2 = _ref53.cleanup;
    var scopeTarget = el;
    if (modifiers.includes("parent")) {
      scopeTarget = el.parentNode;
    }
    var evaluateGet = evaluateLater(scopeTarget, expression);
    var evaluateSet;
    if (typeof expression === "string") {
      evaluateSet = evaluateLater(scopeTarget, "".concat(expression, " = __placeholder"));
    } else if (typeof expression === "function" && typeof expression() === "string") {
      evaluateSet = evaluateLater(scopeTarget, "".concat(expression(), " = __placeholder"));
    } else {
      evaluateSet = function evaluateSet() {};
    }
    var getValue = function getValue() {
      var result;
      evaluateGet(function (value) {
        return result = value;
      });
      return isGetterSetter(result) ? result.get() : result;
    };
    var setValue = function setValue(value) {
      var result;
      evaluateGet(function (value2) {
        return result = value2;
      });
      if (isGetterSetter(result)) {
        result.set(value);
      } else {
        evaluateSet(function () {}, {
          scope: {
            "__placeholder": value
          }
        });
      }
    };
    if (typeof expression === "string" && el.type === "radio") {
      mutateDom(function () {
        if (!el.hasAttribute("name")) el.setAttribute("name", expression);
      });
    }
    var event = el.tagName.toLowerCase() === "select" || ["checkbox", "radio"].includes(el.type) || modifiers.includes("lazy") ? "change" : "input";
    var removeListener = isCloning ? function () {} : on(el, event, modifiers, function (e) {
      setValue(getInputValue(el, modifiers, e, getValue()));
    });
    if (modifiers.includes("fill")) {
      if ([void 0, null, ""].includes(getValue()) || isCheckbox(el) && Array.isArray(getValue()) || el.tagName.toLowerCase() === "select" && el.multiple) {
        setValue(getInputValue(el, modifiers, {
          target: el
        }, getValue()));
      }
    }
    if (!el._x_removeModelListeners) el._x_removeModelListeners = {};
    el._x_removeModelListeners["default"] = removeListener;
    cleanup2(function () {
      return el._x_removeModelListeners["default"]();
    });
    if (el.form) {
      var removeResetListener = on(el.form, "reset", [], function (e) {
        nextTick(function () {
          return el._x_model && el._x_model.set(getInputValue(el, modifiers, {
            target: el
          }, getValue()));
        });
      });
      cleanup2(function () {
        return removeResetListener();
      });
    }
    el._x_model = {
      get: function get() {
        return getValue();
      },
      set: function set(value) {
        setValue(value);
      }
    };
    el._x_forceModelUpdate = function (value) {
      if (value === void 0 && typeof expression === "string" && expression.match(/\./)) value = "";
      window.fromModel = true;
      mutateDom(function () {
        return bind(el, "value", value);
      });
      delete window.fromModel;
    };
    effect3(function () {
      var value = getValue();
      if (modifiers.includes("unintrusive") && document.activeElement.isSameNode(el)) return;
      el._x_forceModelUpdate(value);
    });
  });
  function getInputValue(el, modifiers, event, currentValue) {
    return mutateDom(function () {
      if (event instanceof CustomEvent && event.detail !== void 0) return event.detail !== null && event.detail !== void 0 ? event.detail : event.target.value;else if (isCheckbox(el)) {
        if (Array.isArray(currentValue)) {
          var newValue = null;
          if (modifiers.includes("number")) {
            newValue = safeParseNumber(event.target.value);
          } else if (modifiers.includes("boolean")) {
            newValue = safeParseBoolean(event.target.value);
          } else {
            newValue = event.target.value;
          }
          return event.target.checked ? currentValue.includes(newValue) ? currentValue : currentValue.concat([newValue]) : currentValue.filter(function (el2) {
            return !checkedAttrLooseCompare2(el2, newValue);
          });
        } else {
          return event.target.checked;
        }
      } else if (el.tagName.toLowerCase() === "select" && el.multiple) {
        if (modifiers.includes("number")) {
          return Array.from(event.target.selectedOptions).map(function (option) {
            var rawValue = option.value || option.text;
            return safeParseNumber(rawValue);
          });
        } else if (modifiers.includes("boolean")) {
          return Array.from(event.target.selectedOptions).map(function (option) {
            var rawValue = option.value || option.text;
            return safeParseBoolean(rawValue);
          });
        }
        return Array.from(event.target.selectedOptions).map(function (option) {
          return option.value || option.text;
        });
      } else {
        var _newValue;
        if (isRadio(el)) {
          if (event.target.checked) {
            _newValue = event.target.value;
          } else {
            _newValue = currentValue;
          }
        } else {
          _newValue = event.target.value;
        }
        if (modifiers.includes("number")) {
          return safeParseNumber(_newValue);
        } else if (modifiers.includes("boolean")) {
          return safeParseBoolean(_newValue);
        } else if (modifiers.includes("trim")) {
          return _newValue.trim();
        } else {
          return _newValue;
        }
      }
    });
  }
  function safeParseNumber(rawValue) {
    var number = rawValue ? parseFloat(rawValue) : null;
    return isNumeric2(number) ? number : rawValue;
  }
  function checkedAttrLooseCompare2(valueA, valueB) {
    return valueA == valueB;
  }
  function isNumeric2(subject) {
    return !Array.isArray(subject) && !isNaN(subject);
  }
  function isGetterSetter(value) {
    return value !== null && _typeof(value) === "object" && typeof value.get === "function" && typeof value.set === "function";
  }

  // packages/alpinejs/src/directives/x-cloak.js
  directive("cloak", function (el) {
    return queueMicrotask(function () {
      return mutateDom(function () {
        return el.removeAttribute(prefix("cloak"));
      });
    });
  });

  // packages/alpinejs/src/directives/x-init.js
  addInitSelector(function () {
    return "[".concat(prefix("init"), "]");
  });
  directive("init", skipDuringClone(function (el, _ref54, _ref55) {
    var expression = _ref54.expression;
    var evaluate2 = _ref55.evaluate;
    if (typeof expression === "string") {
      return !!expression.trim() && evaluate2(expression, {}, false);
    }
    return evaluate2(expression, {}, false);
  }));

  // packages/alpinejs/src/directives/x-text.js
  directive("text", function (el, _ref56, _ref57) {
    var expression = _ref56.expression;
    var effect3 = _ref57.effect,
      evaluateLater2 = _ref57.evaluateLater;
    var evaluate2 = evaluateLater2(expression);
    effect3(function () {
      evaluate2(function (value) {
        mutateDom(function () {
          el.textContent = value;
        });
      });
    });
  });

  // packages/alpinejs/src/directives/x-html.js
  directive("html", function (el, _ref58, _ref59) {
    var expression = _ref58.expression;
    var effect3 = _ref59.effect,
      evaluateLater2 = _ref59.evaluateLater;
    var evaluate2 = evaluateLater2(expression);
    effect3(function () {
      evaluate2(function (value) {
        mutateDom(function () {
          el.innerHTML = value;
          el._x_ignoreSelf = true;
          initTree(el);
          delete el._x_ignoreSelf;
        });
      });
    });
  });

  // packages/alpinejs/src/directives/x-bind.js
  mapAttributes(startingWith(":", into(prefix("bind:"))));
  var handler2 = function handler2(el, _ref60, _ref61) {
    var value = _ref60.value,
      modifiers = _ref60.modifiers,
      expression = _ref60.expression,
      original = _ref60.original;
    var effect3 = _ref61.effect,
      cleanup2 = _ref61.cleanup;
    if (!value) {
      var bindingProviders = {};
      injectBindingProviders(bindingProviders);
      var getBindings = evaluateLater(el, expression);
      getBindings(function (bindings) {
        applyBindingsObject(el, bindings, original);
      }, {
        scope: bindingProviders
      });
      return;
    }
    if (value === "key") return storeKeyForXFor(el, expression);
    if (el._x_inlineBindings && el._x_inlineBindings[value] && el._x_inlineBindings[value].extract) {
      return;
    }
    var evaluate2 = evaluateLater(el, expression);
    effect3(function () {
      return evaluate2(function (result) {
        if (result === void 0 && typeof expression === "string" && expression.match(/\./)) {
          result = "";
        }
        mutateDom(function () {
          return bind(el, value, result, modifiers);
        });
      });
    });
    cleanup2(function () {
      el._x_undoAddedClasses && el._x_undoAddedClasses();
      el._x_undoAddedStyles && el._x_undoAddedStyles();
    });
  };
  handler2.inline = function (el, _ref62) {
    var value = _ref62.value,
      modifiers = _ref62.modifiers,
      expression = _ref62.expression;
    if (!value) return;
    if (!el._x_inlineBindings) el._x_inlineBindings = {};
    el._x_inlineBindings[value] = {
      expression: expression,
      extract: false
    };
  };
  directive("bind", handler2);
  function storeKeyForXFor(el, expression) {
    el._x_keyExpression = expression;
  }

  // packages/alpinejs/src/directives/x-data.js
  addRootSelector(function () {
    return "[".concat(prefix("data"), "]");
  });
  directive("data", function (el, _ref63, _ref64) {
    var expression = _ref63.expression;
    var cleanup2 = _ref64.cleanup;
    if (shouldSkipRegisteringDataDuringClone(el)) return;
    expression = expression === "" ? "{}" : expression;
    var magicContext = {};
    injectMagics(magicContext, el);
    var dataProviderContext = {};
    injectDataProviders(dataProviderContext, magicContext);
    var data2 = evaluate(el, expression, {
      scope: dataProviderContext
    });
    if (data2 === void 0 || data2 === true) data2 = {};
    injectMagics(data2, el);
    var reactiveData = reactive(data2);
    initInterceptors(reactiveData);
    var undo = addScopeToNode(el, reactiveData);
    reactiveData["init"] && evaluate(el, reactiveData["init"]);
    cleanup2(function () {
      reactiveData["destroy"] && evaluate(el, reactiveData["destroy"]);
      undo();
    });
  });
  interceptClone(function (from, to) {
    if (from._x_dataStack) {
      to._x_dataStack = from._x_dataStack;
      to.setAttribute("data-has-alpine-state", true);
    }
  });
  function shouldSkipRegisteringDataDuringClone(el) {
    if (!isCloning) return false;
    if (isCloningLegacy) return true;
    return el.hasAttribute("data-has-alpine-state");
  }

  // packages/alpinejs/src/directives/x-show.js
  directive("show", function (el, _ref65, _ref66) {
    var modifiers = _ref65.modifiers,
      expression = _ref65.expression;
    var effect3 = _ref66.effect;
    var evaluate2 = evaluateLater(el, expression);
    if (!el._x_doHide) el._x_doHide = function () {
      mutateDom(function () {
        el.style.setProperty("display", "none", modifiers.includes("important") ? "important" : void 0);
      });
    };
    if (!el._x_doShow) el._x_doShow = function () {
      mutateDom(function () {
        if (el.style.length === 1 && el.style.display === "none") {
          el.removeAttribute("style");
        } else {
          el.style.removeProperty("display");
        }
      });
    };
    var hide = function hide() {
      el._x_doHide();
      el._x_isShown = false;
    };
    var show = function show() {
      el._x_doShow();
      el._x_isShown = true;
    };
    var clickAwayCompatibleShow = function clickAwayCompatibleShow() {
      return setTimeout(show);
    };
    var toggle = once(function (value) {
      return value ? show() : hide();
    }, function (value) {
      if (typeof el._x_toggleAndCascadeWithTransitions === "function") {
        el._x_toggleAndCascadeWithTransitions(el, value, show, hide);
      } else {
        value ? clickAwayCompatibleShow() : hide();
      }
    });
    var oldValue;
    var firstTime = true;
    effect3(function () {
      return evaluate2(function (value) {
        if (!firstTime && value === oldValue) return;
        if (modifiers.includes("immediate")) value ? clickAwayCompatibleShow() : hide();
        toggle(value);
        oldValue = value;
        firstTime = false;
      });
    });
  });

  // packages/alpinejs/src/directives/x-for.js
  directive("for", function (el, _ref67, _ref68) {
    var expression = _ref67.expression;
    var effect3 = _ref68.effect,
      cleanup2 = _ref68.cleanup;
    var iteratorNames = parseForExpression(expression);
    var evaluateItems = evaluateLater(el, iteratorNames.items);
    var evaluateKey = evaluateLater(el,
    // the x-bind:key expression is stored for our use instead of evaluated.
    el._x_keyExpression || "index");
    el._x_prevKeys = [];
    el._x_lookup = {};
    effect3(function () {
      return loop(el, iteratorNames, evaluateItems, evaluateKey);
    });
    cleanup2(function () {
      Object.values(el._x_lookup).forEach(function (el2) {
        return mutateDom(function () {
          destroyTree(el2);
          el2.remove();
        });
      });
      delete el._x_prevKeys;
      delete el._x_lookup;
    });
  });
  function loop(el, iteratorNames, evaluateItems, evaluateKey) {
    var isObject2 = function isObject2(i) {
      return _typeof(i) === "object" && !Array.isArray(i);
    };
    var templateEl = el;
    evaluateItems(function (items) {
      if (isNumeric3(items) && items >= 0) {
        items = Array.from(Array(items).keys(), function (i) {
          return i + 1;
        });
      }
      if (items === void 0) items = [];
      var lookup = el._x_lookup;
      var prevKeys = el._x_prevKeys;
      var scopes = [];
      var keys = [];
      if (isObject2(items)) {
        items = Object.entries(items).map(function (_ref69) {
          var _ref70 = _slicedToArray(_ref69, 2),
            key = _ref70[0],
            value = _ref70[1];
          var scope2 = getIterationScopeVariables(iteratorNames, value, key, items);
          evaluateKey(function (value2) {
            if (keys.includes(value2)) warn("Duplicate key on x-for", el);
            keys.push(value2);
          }, {
            scope: _objectSpread({
              index: key
            }, scope2)
          });
          scopes.push(scope2);
        });
      } else {
        for (var i = 0; i < items.length; i++) {
          var scope2 = getIterationScopeVariables(iteratorNames, items[i], i, items);
          evaluateKey(function (value) {
            if (keys.includes(value)) warn("Duplicate key on x-for", el);
            keys.push(value);
          }, {
            scope: _objectSpread({
              index: i
            }, scope2)
          });
          scopes.push(scope2);
        }
      }
      var adds = [];
      var moves = [];
      var removes = [];
      var sames = [];
      for (var _i2 = 0; _i2 < prevKeys.length; _i2++) {
        var key = prevKeys[_i2];
        if (keys.indexOf(key) === -1) removes.push(key);
      }
      prevKeys = prevKeys.filter(function (key) {
        return !removes.includes(key);
      });
      var lastKey = "template";
      for (var _i3 = 0; _i3 < keys.length; _i3++) {
        var _key5 = keys[_i3];
        var prevIndex = prevKeys.indexOf(_key5);
        if (prevIndex === -1) {
          prevKeys.splice(_i3, 0, _key5);
          adds.push([lastKey, _i3]);
        } else if (prevIndex !== _i3) {
          var keyInSpot = prevKeys.splice(_i3, 1)[0];
          var keyForSpot = prevKeys.splice(prevIndex - 1, 1)[0];
          prevKeys.splice(_i3, 0, keyForSpot);
          prevKeys.splice(prevIndex, 0, keyInSpot);
          moves.push([keyInSpot, keyForSpot]);
        } else {
          sames.push(_key5);
        }
        lastKey = _key5;
      }
      var _loop4 = function _loop4() {
        var key = removes[_i4];
        if (!(key in lookup)) return "continue";
        mutateDom(function () {
          destroyTree(lookup[key]);
          lookup[key].remove();
        });
        delete lookup[key];
      };
      for (var _i4 = 0; _i4 < removes.length; _i4++) {
        var _ret4 = _loop4();
        if (_ret4 === "continue") continue;
      }
      var _loop5 = function _loop5() {
        var _moves$_i = _slicedToArray(moves[_i5], 2),
          keyInSpot = _moves$_i[0],
          keyForSpot = _moves$_i[1];
        var elInSpot = lookup[keyInSpot];
        var elForSpot = lookup[keyForSpot];
        var marker = document.createElement("div");
        mutateDom(function () {
          if (!elForSpot) warn("x-for \":key\" is undefined or invalid", templateEl, keyForSpot, lookup);
          elForSpot.after(marker);
          elInSpot.after(elForSpot);
          elForSpot._x_currentIfEl && elForSpot.after(elForSpot._x_currentIfEl);
          marker.before(elInSpot);
          elInSpot._x_currentIfEl && elInSpot.after(elInSpot._x_currentIfEl);
          marker.remove();
        });
        elForSpot._x_refreshXForScope(scopes[keys.indexOf(keyForSpot)]);
      };
      for (var _i5 = 0; _i5 < moves.length; _i5++) {
        _loop5();
      }
      var _loop6 = function _loop6() {
        var _adds$_i = _slicedToArray(adds[_i6], 2),
          lastKey2 = _adds$_i[0],
          index = _adds$_i[1];
        var lastEl = lastKey2 === "template" ? templateEl : lookup[lastKey2];
        if (lastEl._x_currentIfEl) lastEl = lastEl._x_currentIfEl;
        var scope2 = scopes[index];
        var key = keys[index];
        var clone2 = document.importNode(templateEl.content, true).firstElementChild;
        var reactiveScope = reactive(scope2);
        addScopeToNode(clone2, reactiveScope, templateEl);
        clone2._x_refreshXForScope = function (newScope) {
          Object.entries(newScope).forEach(function (_ref71) {
            var _ref72 = _slicedToArray(_ref71, 2),
              key2 = _ref72[0],
              value = _ref72[1];
            reactiveScope[key2] = value;
          });
        };
        mutateDom(function () {
          lastEl.after(clone2);
          skipDuringClone(function () {
            return initTree(clone2);
          })();
        });
        if (_typeof(key) === "object") {
          warn("x-for key cannot be an object, it must be a string or an integer", templateEl);
        }
        lookup[key] = clone2;
      };
      for (var _i6 = 0; _i6 < adds.length; _i6++) {
        _loop6();
      }
      for (var _i7 = 0; _i7 < sames.length; _i7++) {
        lookup[sames[_i7]]._x_refreshXForScope(scopes[keys.indexOf(sames[_i7])]);
      }
      templateEl._x_prevKeys = keys;
    });
  }
  function parseForExpression(expression) {
    var forIteratorRE = /,([^,\}\]]*)(?:,([^,\}\]]*))?$/;
    var stripParensRE = /^\s*\(|\)\s*$/g;
    var forAliasRE = /([\s\S]*?)\s+(?:in|of)\s+([\s\S]*)/;
    var inMatch = expression.match(forAliasRE);
    if (!inMatch) return;
    var res = {};
    res.items = inMatch[2].trim();
    var item = inMatch[1].replace(stripParensRE, "").trim();
    var iteratorMatch = item.match(forIteratorRE);
    if (iteratorMatch) {
      res.item = item.replace(forIteratorRE, "").trim();
      res.index = iteratorMatch[1].trim();
      if (iteratorMatch[2]) {
        res.collection = iteratorMatch[2].trim();
      }
    } else {
      res.item = item;
    }
    return res;
  }
  function getIterationScopeVariables(iteratorNames, item, index, items) {
    var scopeVariables = {};
    if (/^\[.*\]$/.test(iteratorNames.item) && Array.isArray(item)) {
      var names = iteratorNames.item.replace("[", "").replace("]", "").split(",").map(function (i) {
        return i.trim();
      });
      names.forEach(function (name, i) {
        scopeVariables[name] = item[i];
      });
    } else if (/^\{.*\}$/.test(iteratorNames.item) && !Array.isArray(item) && _typeof(item) === "object") {
      var _names = iteratorNames.item.replace("{", "").replace("}", "").split(",").map(function (i) {
        return i.trim();
      });
      _names.forEach(function (name) {
        scopeVariables[name] = item[name];
      });
    } else {
      scopeVariables[iteratorNames.item] = item;
    }
    if (iteratorNames.index) scopeVariables[iteratorNames.index] = index;
    if (iteratorNames.collection) scopeVariables[iteratorNames.collection] = items;
    return scopeVariables;
  }
  function isNumeric3(subject) {
    return !Array.isArray(subject) && !isNaN(subject);
  }

  // packages/alpinejs/src/directives/x-ref.js
  function handler3() {}
  handler3.inline = function (el, _ref73, _ref74) {
    var expression = _ref73.expression;
    var cleanup2 = _ref74.cleanup;
    var root = closestRoot(el);
    if (!root._x_refs) root._x_refs = {};
    root._x_refs[expression] = el;
    cleanup2(function () {
      return delete root._x_refs[expression];
    });
  };
  directive("ref", handler3);

  // packages/alpinejs/src/directives/x-if.js
  directive("if", function (el, _ref75, _ref76) {
    var expression = _ref75.expression;
    var effect3 = _ref76.effect,
      cleanup2 = _ref76.cleanup;
    if (el.tagName.toLowerCase() !== "template") warn("x-if can only be used on a <template> tag", el);
    var evaluate2 = evaluateLater(el, expression);
    var show = function show() {
      if (el._x_currentIfEl) return el._x_currentIfEl;
      var clone2 = el.content.cloneNode(true).firstElementChild;
      addScopeToNode(clone2, {}, el);
      mutateDom(function () {
        el.after(clone2);
        skipDuringClone(function () {
          return initTree(clone2);
        })();
      });
      el._x_currentIfEl = clone2;
      el._x_undoIf = function () {
        mutateDom(function () {
          destroyTree(clone2);
          clone2.remove();
        });
        delete el._x_currentIfEl;
      };
      return clone2;
    };
    var hide = function hide() {
      if (!el._x_undoIf) return;
      el._x_undoIf();
      delete el._x_undoIf;
    };
    effect3(function () {
      return evaluate2(function (value) {
        value ? show() : hide();
      });
    });
    cleanup2(function () {
      return el._x_undoIf && el._x_undoIf();
    });
  });

  // packages/alpinejs/src/directives/x-id.js
  directive("id", function (el, _ref77, _ref78) {
    var expression = _ref77.expression;
    var evaluate2 = _ref78.evaluate;
    var names = evaluate2(expression);
    names.forEach(function (name) {
      return setIdRoot(el, name);
    });
  });
  interceptClone(function (from, to) {
    if (from._x_ids) {
      to._x_ids = from._x_ids;
    }
  });

  // packages/alpinejs/src/directives/x-on.js
  mapAttributes(startingWith("@", into(prefix("on:"))));
  directive("on", skipDuringClone(function (el, _ref79, _ref80) {
    var value = _ref79.value,
      modifiers = _ref79.modifiers,
      expression = _ref79.expression;
    var cleanup2 = _ref80.cleanup;
    var evaluate2 = expression ? evaluateLater(el, expression) : function () {};
    if (el.tagName.toLowerCase() === "template") {
      if (!el._x_forwardEvents) el._x_forwardEvents = [];
      if (!el._x_forwardEvents.includes(value)) el._x_forwardEvents.push(value);
    }
    var removeListener = on(el, value, modifiers, function (e) {
      evaluate2(function () {}, {
        scope: {
          "$event": e
        },
        params: [e]
      });
    });
    cleanup2(function () {
      return removeListener();
    });
  }));

  // packages/alpinejs/src/directives/index.js
  warnMissingPluginDirective("Collapse", "collapse", "collapse");
  warnMissingPluginDirective("Intersect", "intersect", "intersect");
  warnMissingPluginDirective("Focus", "trap", "focus");
  warnMissingPluginDirective("Mask", "mask", "mask");
  function warnMissingPluginDirective(name, directiveName, slug) {
    directive(directiveName, function (el) {
      return warn("You can't use [x-".concat(directiveName, "] without first installing the \"").concat(name, "\" plugin here: https://alpinejs.dev/plugins/").concat(slug), el);
    });
  }

  // packages/alpinejs/src/index.js
  alpine_default.setEvaluator(normalEvaluator);
  alpine_default.setReactivityEngine({
    reactive: reactive2,
    effect: effect2,
    release: stop,
    raw: toRaw
  });
  var src_default = alpine_default;

  // packages/alpinejs/builds/cdn.js
  window.Alpine = src_default;
  queueMicrotask(function () {
    src_default.start();
  });
})();
