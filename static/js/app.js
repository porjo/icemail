$(function() {
	const apiURL = '//localhost:8080/api';

	const msgLimit = 35;

	const fields = [
		"From",
		"To",
		"Subject"
	];

	const dateFormat = "ddd, DD MMM YYYY HH:mm:ss Z"

	Vue.filter('fromNow', function(value) {
		if( moment.isMoment(value) ) {
			return value.fromNow();
		} else {
			return moment(value, dateFormat).fromNow();
		}
	});

	Vue.filter('formatted', function(value) {
		if( moment.isMoment(value) ) {
			return value.format(dateFormat);
		} else {
			return moment(value, dateFormat).format();
		}
	});

	const Message = Vue.extend({
		template: '#message-view',
		data: function() {
			return {
				modal: {
					title: '',
					body: '',
					id: 0,
					delivered: '',
				}
			}
		},

		watch: {
			'$route': 'viewMsg'
		},

		created: function () {
			this.viewMsg();
		},

		methods: {
			viewMsg: function() {
				var self = this;
				$.get(apiURL + '/search/' + this.$route.params.id, function(data) {
					self.modal.title = data.Emails[0].Header.Subject[0];
					self.modal.body = data.Emails[0].Body;
					self.modal.id = data.Emails[0].ID;
					if( typeof data.Emails[0].Delivered != "undefined") {
						self.modal.delivered = moment(data.Emails[0].Delivered);
					} else {
						self.modal.delivered = '';
					}
				});
			}
		}
	});

	const List = Vue.extend({
		template: '#message-list',

		data: function() {
			return {
				request: this.$parent.request,
				searchDays: 0,
				result: {
					emails: [],
					total: 0,
					offset: 0
				},
				fields: fields
			}
		},

		created: function () {
			this.searchMsg()
		},

		methods: {
			pageActive: function(n) {
				var page = Math.ceil(this.result.offset / this.request.limit) + 1;
				var val = (n == page);
				return {active: val}
			},

			sendMsg: function(id) {
				var self = this;
				$.get(apiURL + '/mail/' + id, function(data) {
				});
			},

			viewMsg: function(id) {
				// ignore click if mouse selected
				var haveSel = getSelection().toString().length > 0;
				if( haveSel ) {
					return;
				}
				router.push({ name: 'message', params: { id: id }});
			},

			toggleSearchOptions: function() {
				$('#search_options').slideToggle();
			},

			searchMsg: function(direction, pageNo) {

				var request = $.extend({}, this.request);

				if( this.searchDays > 0) {
					var startTime = new Date();
					startTime.setDate(startTime.getDate() - this.searchDays);
					request.starttime = startTime.toISOString();
				}

				if( typeof direction !== "undefined" ) {
					var offset = 0;
					if(direction == 'back') {
						offset = this.result.offset - request.limit;
					} else if(direction == 'page') {
						if( typeof pageNo !== "undefined" ) {
							offset = request.limit * (pageNo-1);
						}
					} else {
						offset = this.result.offset + request.limit;
					}
					if( offset < 0  || offset > this.result.total ) {
						// abort search out of limits
						return;
					} else {
						request.offset = offset;
					}
				}

				var self = this;
				$.ajax({
					url: apiURL + '/search',
					type: 'post',
					data: JSON.stringify(request),
					contentType: 'application/json',
					dataType: 'json',
					success: function(data) {
						cleanData(data);
						self.result.emails = data.Emails;
						self.result.total = data.Total;
						self.result.offset = data.Offset;
						self.result.pages = Math.ceil( data.Total / request.limit );
					}
				});
			}
		}
	});

	const routes = [
		{ name: 'home', path: '/', component: List },
		{ name: 'message', path: '/message/:id', component: Message }
	];

	const router = new VueRouter({
		routes
	});

	const App = new Vue({
		router,
		data: {
			request: {
				query: "",
				limit: msgLimit,
				offset: 0,
				locations: fields,
			},
		},
	}).$mount('#app');


	function cleanData(data) {
		$.each(data.Emails, function(k,v) {
			if( typeof v.Header.Date === 'undefined' )
				v.Header.Date = '';
			if( typeof v.Header.From === 'undefined' )
				v.Header.From = '';
			if( typeof v.Header.To === 'undefined' )
				v.Header.To = '';
		});
	}
});
