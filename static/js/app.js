$(function() {
	const apiURL = '//localhost:8080/api';

	const msgLimit = 15;

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
				},
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
				request: {
					query: "",
					limit: msgLimit,
					offset: 0,
					locations: fields,
				},
				searchDays: 0,
				result: {
					emails: [],
					total: 0,
					offset: 0,
					error: ''
				},
				fields: fields
			}
		},

		watch: {
			'$route' (to, from) {
				if(to.fullPath != from.fullPath) {
					this.searchMsg();
				}
			}
		},

		created: function () {
			if('request' in this.$route.params) {
				this.request = this.$route.params.request;
			} else if(this.$route.query.query != '') {
				this.request.query = this.$route.query.query;
			}

			this.searchMsg();
		},

		methods: {

			resetError: function() {
				this.result.error = '';
			},

			currentPage: function() {
				return Math.ceil(this.result.offset / this.request.limit) + 1;
			},

			// used to set active CSS class
			pageActive: function(n) {
				var page = this.currentPage();
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
				this.request.offset = this.result.offset
				router.push({ name: 'message', params: { id: id , request: this.request}});
			},

			toggleSearchOptions: function() {
				$('#search_options').slideToggle();
			},

			searchQuery: function() {
				if(this.request.query != '') {
					if(this.request.query == this.$route.query.query) {
						this.searchMsg();
					} else {
						router.push({ name: 'search', query: {query: this.request.query}});
					}
				} else {
					router.push({ name: 'search', query: {}});
				}
			},

			searchMsg: function() {

				var request = $.extend({}, this.request);

				this.resetError();

				if( this.searchDays > 0) {
					var startTime = new Date();
					startTime.setDate(startTime.getDate() - this.searchDays);
					request.starttime = startTime.toISOString();
				}

				if( 'page' in this.$route.query ) {
					request.offset = request.limit * (this.$route.query.page-1);
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
					},
					error: function (xhr, ajaxOptions, thrownError) {
						self.result.error = xhr.responseText;
					}
				});
			}
		}
	});

	const routes = [
		{ name: 'search', path: '/', component: List },
		{ name: 'message', path: '/message/:id', component: Message }
	];

	const router = new VueRouter({
		routes: routes,
	});

	const App = new Vue({
		router,
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
