/** Polyfill for DOMParser and XMLSerializer
 * DOMParser (String, String)
 *   @source https://gist.github.com/1129031
 * XMLSerializer (Object of HTMLElement)
 */

(function(objects) {
  "use strict";

  var parser     = objects.parser.prototype.parseFromString,
      serializer = objects.serializer.prototype.serializeToString;

  try {
    // Firefox/Opera/IE - throw
    // Webkit - null
    if(
      (new objects.parser()).parseFromString("", "text/html")
&&    (new objects.serializer()).serializeToString(document.body)
    ) return;
  } catch(error) {}

  objects.parser.prototype.parseFromString = function(markup, type) {
    if(/^\s*text\/html?\s*(?:;|$)/i.test(type)) {
      var chronicle = document.implementation.createHTMLDocument("");

      if(/<!doctype\b/i.test(markup))
        chronicle.documentElement.innerHTML = markup;
      else
        chronicle.body.innerHTML = markup;
      return chronicle;
    } else {
      return parser.apply(this, arguments);
    }
  };

  objects.serializer.prototype.serializeToString = function(node) {
    var element = node.documentElement || node;
    return (element.outerHTML)?
      element.outerHTML:
    (element.innerHTML)?
      (function(e) {
        var A = e.attributes, S = [], N = e.tagName;
        for(var a in A)
          S.push(a + '="' + A[a].value + '"');
        return '<' + N + (S.length? ' ' + S.join(' '): '') + '>' + e.innerHTML + (/\b(area|base|[bh]r|col|command|embed|img|input|keygen|link|menuitem|meta|param|source|track|wbr)\b/i.test(N)? '': '</' + N + '>')
      })(element):
    "";
  };
})({ parser: DOMParser, serializer: XMLSerializer });

