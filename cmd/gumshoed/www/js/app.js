(function() {
	var app = angular.module('gumshoe', ['fundoo.services']);

  app.controller('ConfigController', ['$log', '$scope', function($log, $scope){
    var setCtrl = this;
    setCtrl.current = {};
    setCtrl.updates = [];
    setCtrl.preventNav = false;

    $.getJSON("/settings")
      .done(function( json ) {
        setCtrl.current = json;
        $scope.user_dir = json.dir_options["user_dir"];
        $scope.fetch_dir = json.dir_options["fetch_dir"];
        $scope.download_dir = json.dir_options["download_dir"];
        $scope.log_dir = json.dir_options["log_dir"];
        $scope.ops = json.operations;
        $scope.tracker = json.download_params
        $scope.irc = json.irc_channel
      });

    this.UpdateConfig = function() {
      if ($scope.$pristine) {
        $log.log("Nothing to update.");
        return
      }
      var sNew = setCtrl.current;
      sNew.download_params = $scope.tracker;
      sNew.operations = $scope.ops;
      sNew.irc_channel = $scope.irc;
      sNew.dir_options["user_dir"] = $scope.user_dir;
      sNew.dir_options["fetch_dir"] = $scope.fetch_dir;
      sNew.dir_options["download_dir"] = $scope.download_dir;
      sNew.dir_options["log_dir"] = $scope.log_dir;

      $log.log('Sending update now.');
      $.post({
        url: "/api/config/update",
        contentType: "application/json",
        async: true,
        data: JSON.stringify(sNew),
        success: function() {
          this.getElementsByName("update_msg").hidden = false;
          setCtrl.updates = [];
        },
      });
    };

    this.ReloadConfig = function() {
      setCtrl.updates = [];
    };

    this.UpdateEvent = function($event) {
      $scope.preventNav = true;
      setCtrl.updates.push($event.ngModel);
    };

    this.ShowHint = function($event) {
      // Raise tooltip box with text from a help file.
      return;
    };

    $scope.$watch('gumshoe-tracker-settings', function(newValue, oldValue) {
      $scope.cPendingBB = true;
      // Do other things here to ensure settings get saved appropriately
    });
  }]);

  app.controller('ShowController', ['$log', '$http', function($log, $http){
    var showCtrl = this;
    showCtrl.shows = [];
    showCtrl.newShow = {};
    var showAddForm = false;

    $http.get("/api/shows").success(function(data){
      showCtrl.shows = data;
    });

    this.boolConv = function(str) {
      switch(str) {
        case "true":
          return true;
        case "false":
          return false;
      }
    };

    this.addShow = function() {
      this.newShow.episodal = this.boolConv(this.newShow.episodal);
      $http.post("/api/show/new", this.newShow).success(function(data){
        showCtrl.newShow = {};
        showCtrl.showAddForm = false;
        showCtrl.shows.push(data);
      }).error(function(data, status, headers, config){
        $log.log(data, status, headers, config);
      });
    };

    this.showEditForm = function(index) {
      if(typeof showCtrl.shows[index].showEditForm == "undefined") {
        showCtrl.shows[index].showEditForm = true;
      } else {
        showCtrl.shows[index].showEditForm = !showCtrl.shows[index].showEditForm;
      }
    };

    this.isEditFormVisible = function(index) {
      if(typeof showCtrl.shows[index].showEditForm == "undefined") {
        showCtrl.shows[index].showEditForm = false;
      }
      return showCtrl.shows[index].showEditForm;
    };

    this.editShow = function(index) {
      newShow = this.shows[index];
      newShow.episodal = this.boolConv(newShow.episodal);
      $http.post("/api/show/update/" + newShow.ID, newShow).success(function(){
        showCtrl.showEditForm(index);
      }).error(function(data, status, headers, config){
        $log.log(data, status, headers, config);
      });
    };

    this.deleteShow = function(index) {
      title = showCtrl.shows[index].title;
      if(window.confirm("Delete " + title + "?")) {
        $http.delete("/api/show/delete/" + showCtrl.shows[index].ID).success(function(data){
          showCtrl.shows.splice(index, 1);
        });
      };
    };

  } ] );

  app.directive("gumshoeTabs", function() {
     return {
       restrict: "E",
       templateUrl: "gumshoe-tabs.html",
       controller: function() {
         this.tab = 1;

         this.isSet = function(checkTab) {
           return this.tab === checkTab;
         };

         this.setTab = function(activeTab) {
           this.tab = activeTab;
         };
       },
       controllerAs: "tab"
     };
   });

  app.directive("gumshoeSettings", function() {
    return {
      restrict: "E",
      templateUrl: "gumshoe-settings.html",
      controller: function() {
        this.sTab = 1;

        this.isSet = function(checkTab) {
          return this.sTab === checkTab;
        };

        this.setTab = function(activeTab) {
          // Check for changes before navigatingaway from tab.
          this.sTab = activeTab;
        };
      },
      controllerAs: "sTab"
    };
  });

	app.directive("gumshoeStatus", function() {
		return {
      restrict: 'E',
      templateUrl: "gumshoe-status.html"
    };
  });
	app.directive("gumshoeShows", function() {
		return {
      restrict: 'E',
      templateUrl: "gumshoe-shows.html"
    };
  });
	app.directive("gumshoeQueue", function() {
		return {
      restrict: 'E',
      templateUrl: "gumshoe-queue.html"
    };
  });
  app.directive("gumshoeBasicSettings", function() {
    return {
      restrict: 'E',
      templateUrl: "setting-basics.html"
    };
  });
  app.directive("settingTracker", function() {
    return {
      restrict: 'E',
      templateUrl: "setting-tracker.html"
    };
  });
})();
