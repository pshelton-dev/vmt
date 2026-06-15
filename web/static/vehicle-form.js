// Populates the make/model comboboxes on the vehicle form from a bundled
// dataset. Both inputs use <datalist>, so values not in the list can still be
// typed freely. Model suggestions are filtered by the currently entered make.
(function () {
  "use strict";

  var makeInput = document.getElementById("make-input");
  var modelInput = document.getElementById("model-input");
  var makesList = document.getElementById("makes-list");
  var modelsList = document.getElementById("models-list");
  if (!makeInput || !modelInput || !makesList || !modelsList) {
    return;
  }

  function fill(listEl, values) {
    listEl.textContent = "";
    var frag = document.createDocumentFragment();
    values.forEach(function (v) {
      var opt = document.createElement("option");
      opt.value = v;
      frag.appendChild(opt);
    });
    listEl.appendChild(frag);
  }

  fetch("/static/vehicle-data.json")
    .then(function (r) {
      return r.ok ? r.json() : {};
    })
    .then(function (data) {
      var makes = Object.keys(data).sort();
      fill(makesList, makes);

      // Case-insensitive lookup so a typed make still matches a known key.
      var byLower = {};
      makes.forEach(function (m) {
        byLower[m.toLowerCase()] = data[m];
      });

      function refreshModels() {
        var models = byLower[makeInput.value.trim().toLowerCase()] || [];
        fill(modelsList, models);
      }

      makeInput.addEventListener("input", refreshModels);
      makeInput.addEventListener("change", refreshModels);
      refreshModels(); // prime for the edit page (make may be pre-filled)
    })
    .catch(function () {
      /* offline / fetch failed — inputs still work as plain text fields */
    });
})();
